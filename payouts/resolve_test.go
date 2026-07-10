package payouts

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/etclabscore/open-etc-pool/rpc"
	"github.com/etclabscore/open-etc-pool/storage"
)

func resolveBackend(t *testing.T) *storage.RedisClient {
	t.Helper()
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, payerPrefix)
	if _, err := backend.Check(); err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}
	ctx := context.Background()
	if keys, _ := backend.Client().Keys(ctx, payerPrefix+":*").Result(); len(keys) > 0 {
		backend.Client().Del(ctx, keys...)
	}
	return backend
}

// seedStuckPayout reproduces the state a payout leaves after it locks, debits the
// balance, and (when txHash != "") broadcasts the tx, then crashes before
// WritePayment records it.
func seedStuckPayout(t *testing.T, backend *storage.RedisClient, login string, amount int64, txHash string) {
	t.Helper()
	ctx := context.Background()
	if err := backend.Client().HSet(ctx, payerPrefix+":miners:"+login, "balance", amount).Err(); err != nil {
		t.Fatalf("seed balance: %v", err)
	}
	if err := backend.LockPayouts(login, amount); err != nil {
		t.Fatalf("LockPayouts: %v", err)
	}
	if err := backend.UpdateBalance(login, amount); err != nil { // balance -> 0, pending -> amount
		t.Fatalf("UpdateBalance: %v", err)
	}
	if txHash != "" {
		if err := backend.SetPendingPaymentTx(login, amount, txHash); err != nil {
			t.Fatalf("SetPendingPaymentTx: %v", err)
		}
	}
}

func minerField(t *testing.T, backend *storage.RedisClient, login, field string) int64 {
	t.Helper()
	v, err := backend.Client().HGet(context.Background(), payerPrefix+":miners:"+login, field).Int64()
	if err == redis.Nil {
		return 0
	}
	if err != nil {
		t.Fatalf("HGet %s: %v", field, err)
	}
	return v
}

// A payout can leave the lock set with no pending-payment records (e.g. it
// crashed between locking and recording the debit). resolvePayouts must clear
// that stuck lock; otherwise every future payout stays blocked.
func TestResolvePayoutsClearsStuckLockWithNoPendingPayments(t *testing.T) {
	backend := resolveBackend(t)

	if err := backend.LockPayouts("0xaa", 5); err != nil {
		t.Fatalf("LockPayouts: %v", err)
	}
	if p := backend.GetPendingPayments(); len(p) != 0 {
		t.Fatalf("expected no pending payments, got %d", len(p))
	}

	proc := &PayoutsProcessor{config: &PayoutsConfig{BgSave: false}, backend: backend}
	proc.resolvePayouts()

	if locked, _ := backend.IsPayoutsLocked(); locked {
		t.Fatal("resolvePayouts must clear the lock even with no pending payments")
	}
}

// A payout that never broadcast its tx (crash before/at send) leaves no tx hash;
// the balance is safe to credit back.
func TestResolvePayoutsCreditsBackWhenNeverBroadcast(t *testing.T) {
	backend := resolveBackend(t)
	const login, amount = "0xbb", int64(500)
	seedStuckPayout(t, backend, login, amount, "")

	proc := &PayoutsProcessor{config: &PayoutsConfig{BgSave: false}, backend: backend, rpc: &fakePayerRPC{}}
	proc.resolvePayouts()

	if bal, _ := backend.GetBalance(login); bal != amount {
		t.Fatalf("balance = %d, want %d credited back", bal, amount)
	}
	if p := backend.GetPendingPayments(); len(p) != 0 {
		t.Fatalf("pending not cleared: %d", len(p))
	}
	if locked, _ := backend.IsPayoutsLocked(); locked {
		t.Fatal("lock not cleared")
	}
}

// A payout whose tx was already broadcast (tx hash recorded) must NOT be credited
// back — that would double-pay. With the tx unverifiable/pending it is recorded
// as paid.
func TestResolvePayoutsDoesNotCreditBackAlreadyBroadcast(t *testing.T) {
	backend := resolveBackend(t)
	const login, amount = "0xcc", int64(700)
	seedStuckPayout(t, backend, login, amount, "0xdeadbeef")

	// nil receipt: tx not yet mined / unverifiable -> treat as paid, don't credit back.
	proc := &PayoutsProcessor{config: &PayoutsConfig{BgSave: false}, backend: backend, rpc: &fakePayerRPC{receipt: nil}}
	proc.resolvePayouts()

	if bal, _ := backend.GetBalance(login); bal != 0 {
		t.Fatalf("balance = %d, want 0 (an already-sent payout must not be credited back)", bal)
	}
	if paid := minerField(t, backend, login, "paid"); paid != amount {
		t.Fatalf("paid = %d, want %d (payout recorded as paid)", paid, amount)
	}
	if p := backend.GetPendingPayments(); len(p) != 0 {
		t.Fatalf("pending not cleared: %d", len(p))
	}
	if locked, _ := backend.IsPayoutsLocked(); locked {
		t.Fatal("lock not cleared")
	}
}

// A payout whose tx was broadcast but provably reverted on-chain moved no value,
// so the balance is credited back.
func TestResolvePayoutsCreditsBackWhenTxReverted(t *testing.T) {
	backend := resolveBackend(t)
	const login, amount = "0xdd", int64(900)
	seedStuckPayout(t, backend, login, amount, "0xreverted")

	reverted := &rpc.TxReceipt{TxHash: "0xreverted", BlockHash: "0xblock", Status: "0x0"} // confirmed, failed
	proc := &PayoutsProcessor{config: &PayoutsConfig{BgSave: false}, backend: backend, rpc: &fakePayerRPC{receipt: reverted}}
	proc.resolvePayouts()

	if bal, _ := backend.GetBalance(login); bal != amount {
		t.Fatalf("balance = %d, want %d credited back after a reverted payout", bal, amount)
	}
	if p := backend.GetPendingPayments(); len(p) != 0 {
		t.Fatalf("pending not cleared: %d", len(p))
	}
}
