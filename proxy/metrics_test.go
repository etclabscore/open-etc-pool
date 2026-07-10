package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/etclabscore/go-etchash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/etclabscore/open-etc-pool/metrics"
	"github.com/etclabscore/open-etc-pool/storage"
)

// These tests drive processShare through each outcome and assert the Prometheus
// counters move by exactly one. They need the Redis service (they skip without
// it). Counters are global, so the assertions use before/after deltas.
//
// The mock reports height 8: it stays in Etchash epoch 0 (small, fast cache),
// but avoids checkPoWExist's height-8 sweep underflowing at tiny heights, which
// would otherwise drop the entry and hide duplicates.

const metricsHeight = uint64(8)

func shareCount(status string) float64 {
	return testutil.ToFloat64(metrics.ShareOutcomes.WithLabelValues(status))
}

func assertShareDelta(t *testing.T, status string, want float64, fn func()) {
	t.Helper()
	before := shareCount(status)
	fn()
	if got := shareCount(status) - before; got != want {
		t.Fatalf("shares{status=%q} delta = %v, want %v", status, got, want)
	}
}

func flushKeys(b *storage.RedisClient, prefix string) {
	ctx := context.Background()
	if keys, _ := b.Client().Keys(ctx, prefix+":*").Result(); len(keys) > 0 {
		b.Client().Del(ctx, keys...)
	}
}

func mineShare(t *testing.T, nonce uint64) (nonceHex, mixHex string) {
	t.Helper()
	m := etchash.New(func() *uint64 { f := e2eFBlock; return &f }(), nil)
	mix, _ := m.Compute(metricsHeight, common.HexToHash(e2eHeader), nonce)
	return fmt.Sprintf("0x%016x", nonce), strings.ToLower(mix.Hex())
}

// startMetricsMock serves fixed work at height 8 with the given getWork target.
// A huge target (e2eWorkTarget) yields shares only; a tiny difficulty (0xff..ff
// target) makes a valid share also a valid block.
func startMetricsMock(target string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		var result interface{}
		switch req.Method {
		case "eth_getWork":
			result = []string{e2eHeader, e2eSeed, target}
		case "eth_getBlockByNumber":
			result = map[string]string{"number": "0x8", "difficulty": "0x1"}
		case "eth_submitWork":
			result = true
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 0, "jsonrpc": "2.0", "result": result})
	}))
}

func newMetricsProxy(t *testing.T, coin, target string) (*ProxyServer, *BlockTemplate) {
	t.Helper()
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, coin)
	if _, err := backend.Check(); err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}
	flushKeys(backend, coin)

	node := startMetricsMock(target)
	t.Cleanup(node.Close)

	cfg := e2eConfig(node.URL)
	cfg.Coin = coin
	cfg.Proxy.Stratum.Enabled = false // no listeners needed; call processShare directly
	proxy := NewProxy(cfg, backend)
	tpl := proxy.currentBlockTemplate()
	if tpl == nil || tpl.Header != e2eHeader {
		t.Fatalf("unexpected block template: %+v", tpl)
	}
	return proxy, tpl
}

func TestShareOutcomeMetrics(t *testing.T) {
	proxy, tpl := newMetricsProxy(t, "testmetrics", e2eWorkTarget)
	const ip = "127.0.0.1"

	// Valid share.
	n0, m0 := mineShare(t, 0)
	assertShareDelta(t, metrics.ShareValid, 1, func() {
		proxy.processShare(e2eLogin, "rig1", ip, tpl, []string{n0, e2eHeader, m0})
	})

	// Duplicate — the same nonce again.
	assertShareDelta(t, metrics.ShareDuplicate, 1, func() {
		proxy.processShare(e2eLogin, "rig1", ip, tpl, []string{n0, e2eHeader, m0})
	})

	// Stale — a header the template doesn't know.
	n1, m1 := mineShare(t, 1)
	otherHeader := "0x" + strings.Repeat("cd", 32)
	assertShareDelta(t, metrics.ShareStale, 1, func() {
		proxy.processShare(e2eLogin, "rig1", ip, tpl, []string{n1, otherHeader, m1})
	})

	// Invalid — right header, wrong mix digest.
	zeroMix := "0x" + strings.Repeat("00", 32)
	assertShareDelta(t, metrics.ShareInvalid, 1, func() {
		proxy.processShare(e2eLogin, "rig1", ip, tpl, []string{"0x0000000000000009", e2eHeader, zeroMix})
	})
}

func TestBlockFoundMetrics(t *testing.T) {
	easyTarget := "0x" + strings.Repeat("f", 64) // network difficulty ~1
	proxy, tpl := newMetricsProxy(t, "testmetricsblk", easyTarget)

	beforeBlocks := testutil.ToFloat64(metrics.BlocksFound)
	beforeAccepted := testutil.ToFloat64(metrics.BlockSubmissions.WithLabelValues(metrics.SubmitAccepted))

	n0, m0 := mineShare(t, 0)
	exist, valid := proxy.processShare(e2eLogin, "rig1", "127.0.0.1", tpl, []string{n0, e2eHeader, m0})
	if exist || !valid {
		t.Fatalf("processShare = (exist=%v, valid=%v), want (false, true)", exist, valid)
	}
	if got := testutil.ToFloat64(metrics.BlocksFound) - beforeBlocks; got != 1 {
		t.Fatalf("blocks_found delta = %v, want 1", got)
	}
	if got := testutil.ToFloat64(metrics.BlockSubmissions.WithLabelValues(metrics.SubmitAccepted)) - beforeAccepted; got != 1 {
		t.Fatalf("block_submissions{accepted} delta = %v, want 1", got)
	}
}
