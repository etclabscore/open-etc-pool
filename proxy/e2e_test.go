package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/etclabscore/go-etchash"
	"github.com/ethereum/go-ethereum/common"

	"github.com/etclabscore/open-etc-pool/policy"
	"github.com/etclabscore/open-etc-pool/storage"
)

// End-to-end mining test: a mock node serves fixed low-difficulty work, the real
// pool runs against it over real sockets, and a test-miner computes a valid
// Etchash share (with the same go-etchash library the pool verifies with) and
// submits it through BOTH ingress protocols — HTTP getwork and Stratum. It
// asserts each share is accepted and lands in Redis.
//
// It fits GitHub Actions because there is no real node (mocked) and the PoW is
// trivial: block height 1 → epoch 0 (~small cache), pool difficulty 1 → any
// nonce is a valid share. Requires the Redis service (skips if unavailable).

const (
	e2eHeader      = "0x" + "abababababababababababababababababababababababababababababababab"
	e2eSeed        = "0x" + "0000000000000000000000000000000000000000000000000000000000000000"
	e2eWorkTarget  = "0x01" // huge network difficulty -> shares, never blocks
	e2eBlockNumber = "0x1"  // height 1 => epoch 0
	e2eLogin       = "0x00000000000000000000000000000000000000aa"
	e2eHTTPAddr    = "127.0.0.1:39888"
	e2eStratumAddr = "127.0.0.1:39008"
	e2eFBlock      = uint64(11700000) // ecip1099FBlockClassic
	e2ePrefix      = "teste2e"
)

// startMockNode serves just enough JSON-RPC for the pool: getWork (also the
// health check), the pending block, and submitWork.
func startMockNode() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		var result interface{}
		switch req.Method {
		case "eth_getWork":
			result = []string{e2eHeader, e2eSeed, e2eWorkTarget}
		case "eth_getBlockByNumber":
			result = map[string]string{"number": e2eBlockNumber, "difficulty": "0x1"}
		case "eth_submitWork":
			result = true
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 0, "jsonrpc": "2.0", "result": result})
	}))
}

func e2eConfig(upstreamURL string) *Config {
	return &Config{
		Name:    "test",
		Network: "classic",
		Coin:    e2ePrefix,
		Redis:   storage.Config{Endpoint: "127.0.0.1:6379"},
		Upstream: []Upstream{
			{Name: "mock", Url: upstreamURL, Timeout: "10s"},
		},
		UpstreamCheckInterval: "5s",
		Proxy: Proxy{
			Enabled:              true,
			Listen:               e2eHTTPAddr,
			LimitHeadersSize:     1 << 10,
			LimitBodySize:        1 << 20,
			BlockRefreshInterval: "1s",
			StateUpdateInterval:  "3s",
			Difficulty:           1, // any nonce is a valid share
			HashrateExpiration:   "3h",
			HealthCheck:          true,
			MaxFails:             100,
			Stratum: Stratum{
				Enabled: true,
				Listen:  e2eStratumAddr,
				Timeout: "60s",
				MaxConn: 128,
			},
			Policy: policy.Config{
				Workers:         8,
				ResetInterval:   "60m",
				RefreshInterval: "60m",
				Banning:         policy.Banning{Enabled: false},
				Limits:          policy.Limits{Enabled: false, Grace: "5m"},
			},
		},
	}
}

func waitForPort(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("listener %s never came up", addr)
}

func TestEndToEndMining(t *testing.T) {
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, e2ePrefix)
	if _, err := backend.Check(); err != nil {
		t.Skipf("Redis not available, skipping e2e: %v", err)
	}
	ctx := context.Background()
	if keys, _ := backend.Client().Keys(ctx, e2ePrefix+":*").Result(); len(keys) > 0 {
		backend.Client().Del(ctx, keys...)
	}

	node := startMockNode()
	defer node.Close()

	proxy := NewProxy(e2eConfig(node.URL), backend) // fetches template + starts stratum listener
	go proxy.Start()                                // HTTP listener
	waitForPort(t, e2eHTTPAddr)
	waitForPort(t, e2eStratumAddr)

	// The test-miner uses the same library and epoch the pool verifies with.
	miner := etchash.New(func() *uint64 { f := e2eFBlock; return &f }(), nil)
	compute := func(nonce uint64) (nonceHex, mixHex string) {
		mix, _ := miner.Compute(1, common.HexToHash(e2eHeader), nonce)
		return fmt.Sprintf("0x%016x", nonce), strings.ToLower(mix.Hex())
	}

	// --- Ingress 1: HTTP getwork ---
	httpURL := "http://" + e2eHTTPAddr + "/" + e2eLogin + "/rig1"
	work := httpGetWork(t, httpURL)
	if len(work) < 3 || work[0] != e2eHeader {
		t.Fatalf("HTTP getwork returned unexpected work: %v", work)
	}
	nonceHex, mixHex := compute(0)
	if !httpSubmit(t, httpURL, nonceHex, e2eHeader, mixHex) {
		t.Fatal("HTTP getwork share was rejected")
	}
	t.Log("HTTP getwork share accepted")

	// --- Ingress 2: Stratum ---
	nonceHex, mixHex = compute(1) // a different nonce so it isn't a duplicate
	if !stratumMine(t, e2eStratumAddr, nonceHex, mixHex) {
		t.Fatal("Stratum share was rejected")
	}
	t.Log("Stratum share accepted")

	// --- Both shares must have reached Redis ---
	stats, err := backend.GetMinerStats(e2eLogin, 30)
	if err != nil {
		t.Fatalf("GetMinerStats: %v", err)
	}
	if rs, _ := stats["roundShares"].(int64); rs != 2 {
		t.Fatalf("round shares in Redis = %v, want 2 (one per ingress)", stats["roundShares"])
	}
}

func httpGetWork(t *testing.T, url string) []string {
	t.Helper()
	raw := httpRPC(t, url, "eth_getWork", []string{})
	var work []string
	if err := json.Unmarshal(raw, &work); err != nil {
		t.Fatalf("decode getWork result %s: %v", raw, err)
	}
	return work
}

func httpSubmit(t *testing.T, url, nonce, header, mix string) bool {
	t.Helper()
	raw := httpRPC(t, url, "eth_submitWork", []string{nonce, header, mix})
	var ok bool
	_ = json.Unmarshal(raw, &ok)
	return ok
}

func httpRPC(t *testing.T, url, method string, params interface{}) json.RawMessage {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", method, err)
	}
	defer resp.Body.Close()
	var out struct {
		Result json.RawMessage `json:"result"`
		Error  interface{}     `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s response: %v", method, err)
	}
	if out.Error != nil {
		t.Fatalf("%s returned error: %v", method, out.Error)
	}
	return out.Result
}

func stratumMine(t *testing.T, addr, nonce, mix string) bool {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("stratum dial: %v", err)
	}
	defer conn.Close()
	br := bufio.NewReader(conn)

	call := func(id int, method string, params []string) json.RawMessage {
		req, _ := json.Marshal(map[string]interface{}{"id": id, "method": method, "params": params, "worker": "rig1"})
		conn.SetDeadline(time.Now().Add(60 * time.Second))
		if _, err := conn.Write(append(req, '\n')); err != nil {
			t.Fatalf("stratum write %s: %v", method, err)
		}
		line, err := br.ReadBytes('\n')
		if err != nil {
			t.Fatalf("stratum read %s: %v", method, err)
		}
		var out struct {
			Result json.RawMessage `json:"result"`
			Error  interface{}     `json:"error"`
		}
		_ = json.Unmarshal(line, &out)
		if out.Error != nil {
			t.Fatalf("stratum %s error: %v", method, out.Error)
		}
		return out.Result
	}

	var loggedIn bool
	_ = json.Unmarshal(call(1, "eth_submitLogin", []string{e2eLogin, "x"}), &loggedIn)
	if !loggedIn {
		t.Fatal("stratum login failed")
	}
	work := call(2, "eth_getWork", nil)
	var w []string
	if err := json.Unmarshal(work, &w); err != nil || len(w) < 3 || w[0] != e2eHeader {
		t.Fatalf("stratum getWork unexpected: %s", work)
	}
	var ok bool
	_ = json.Unmarshal(call(3, "eth_submitWork", []string{nonce, e2eHeader, mix}), &ok)
	return ok
}
