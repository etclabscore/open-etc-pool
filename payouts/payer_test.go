package payouts

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/etclabscore/open-etc-pool/rpc"
	"github.com/etclabscore/open-etc-pool/storage"
)

type fakePayerRPC struct {
	peers          int64
	signErr        error
	poolBalance    *big.Int
	poolBalanceErr error
	sendTxErr      error
	receipt        *rpc.TxReceipt
	receiptErr     error
	txCountLatest  uint64
	txCountPending uint64
	txCountErr     error
}

func (f *fakePayerRPC) GetPeerCount() (int64, error)         { return f.peers, nil }
func (f *fakePayerRPC) Sign(from, s string) (string, error)  { return "0xsig", f.signErr }
func (f *fakePayerRPC) GetBalance(a string) (*big.Int, error) { return f.poolBalance, f.poolBalanceErr }
func (f *fakePayerRPC) SendTransaction(from, to, gas, gasPrice, value string, autoGas bool) (string, error) {
	return "0xtxhash", f.sendTxErr
}
func (f *fakePayerRPC) GetTxReceipt(hash string) (*rpc.TxReceipt, error) { return f.receipt, f.receiptErr }
func (f *fakePayerRPC) GetTxCount(addr, tag string) (uint64, error) {
	if f.txCountErr != nil {
		return 0, f.txCountErr
	}
	if tag == "pending" {
		return f.txCountPending, nil
	}
	return f.txCountLatest, nil
}

const payerPrefix = "test-payer"

func newTestPayer(t *testing.T, fake payerRPC) *PayoutsProcessor {
	t.Helper()
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, payerPrefix)

	ctx := context.Background()
	if keys, _ := backend.Client().Keys(ctx, payerPrefix+":*").Result(); len(keys) > 0 {
		backend.Client().Del(ctx, keys...)
	}
	// One payee above the payout threshold.
	if err := backend.Client().HSet(ctx, payerPrefix+":miners:0xaa", "balance", "1000").Err(); err != nil {
		t.Fatalf("seed miner: %v", err)
	}

	u := NewPayoutsProcessor(&PayoutsConfig{
		Interval:     "1h",
		RequirePeers: 1,
		Threshold:    100,
		Address:      "0x0000000000000000000000000000000000000001",
		Gas:          "21000",
		GasPrice:     "50000000000",
		Daemon:       "http://127.0.0.1:1",
		Timeout:      "1s",
	}, backend)
	u.rpc = fake
	return u
}

// A transient error BEFORE any state mutation (fetching the pool balance) must
// retry next cycle, not permanently halt payouts.
func TestPayerRetriesOnPreMutationError(t *testing.T) {
	fake := &fakePayerRPC{peers: 25, poolBalanceErr: errors.New("node unreachable")}
	u := newTestPayer(t, fake)

	u.process()

	if u.halt {
		t.Error("a pre-mutation RPC error must not halt payouts")
	}
}

// A failure AFTER state has been mutated (the transaction was sent) must halt
// and require manual resolution, to avoid double-paying.
func TestPayerHaltsOnPostMutationError(t *testing.T) {
	fake := &fakePayerRPC{
		peers:       25,
		poolBalance: big.NewInt(1000000000000000000), // 1 ETC in Wei, plenty
		sendTxErr:   errors.New("broadcast failed"),
	}
	u := newTestPayer(t, fake)

	u.process()

	if !u.halt {
		t.Error("a post-mutation error (SendTransaction) must halt payouts (safety latch)")
	}
}
