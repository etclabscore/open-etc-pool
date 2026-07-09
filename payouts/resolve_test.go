package payouts

import (
	"context"
	"testing"

	"github.com/etclabscore/open-etc-pool/storage"
)

// A payout can leave the lock set with no pending-payment records (e.g. it
// crashed between locking and recording the debit). resolvePayouts must clear
// that stuck lock; otherwise every future payout stays blocked with no way to
// recover short of manual Redis surgery.
func TestResolvePayoutsClearsStuckLockWithNoPendingPayments(t *testing.T) {
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, payerPrefix)
	if _, err := backend.Check(); err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}
	ctx := context.Background()
	if keys, _ := backend.Client().Keys(ctx, payerPrefix+":*").Result(); len(keys) > 0 {
		backend.Client().Del(ctx, keys...)
	}

	if err := backend.LockPayouts("0xaa", 5); err != nil {
		t.Fatalf("LockPayouts: %v", err)
	}
	if locked, _ := backend.IsPayoutsLocked(); !locked {
		t.Fatal("payouts should be locked after LockPayouts")
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
