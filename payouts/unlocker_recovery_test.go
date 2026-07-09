package payouts

import (
	"errors"
	"testing"

	"github.com/etclabscore/open-etc-pool/rpc"
	"github.com/etclabscore/open-etc-pool/storage"
)

// fakeUnlockerRPC lets a test fail GetPendingBlock on demand. GetPendingBlock is
// the first RPC of a cycle, so failing it exercises the halt/recover path
// without reaching Redis.
type fakeUnlockerRPC struct {
	pendingErr error
}

func (f *fakeUnlockerRPC) GetPendingBlock() (*rpc.GetBlockReplyPart, error) {
	if f.pendingErr != nil {
		return nil, f.pendingErr
	}
	return &rpc.GetBlockReplyPart{Number: "0x1"}, nil
}

func (f *fakeUnlockerRPC) GetBlockByHeight(int64) (*rpc.GetBlockReply, error) { return nil, nil }

func (f *fakeUnlockerRPC) GetUncleByBlockNumberAndIndex(int64, int) (*rpc.GetBlockReply, error) {
	return nil, nil
}

func (f *fakeUnlockerRPC) GetTxReceipt(string) (*rpc.TxReceipt, error) { return nil, nil }

func newTestUnlocker(rpcClient unlockerRPC) *BlockUnlocker {
	network := "classic"
	backend := storage.NewRedisClient(&storage.Config{Endpoint: "127.0.0.1:6379"}, "test-unlock-recover")
	u := NewBlockUnlocker(&UnlockerConfig{
		Interval:      "1h",
		Depth:         120,
		ImmatureDepth: 20,
		Daemon:        "http://127.0.0.1:1",
		Timeout:       "1s",
	}, backend, &network)
	u.rpc = rpcClient
	return u
}

// A transient RPC error suspends the cycle, but the next cycle recovers once the
// node is reachable again.
func TestUnlockerRecoversFromTransientError(t *testing.T) {
	fake := &fakeUnlockerRPC{}
	u := newTestUnlocker(fake)

	fake.pendingErr = errors.New("node unreachable")
	u.runCycle()
	if !u.halt {
		t.Fatal("halt must be set after a failing cycle")
	}
	if u.failsInARow != 1 {
		t.Fatalf("failsInARow = %d, want 1", u.failsInARow)
	}

	fake.pendingErr = nil // node is back
	u.runCycle()
	if u.halt {
		t.Fatal("halt must clear after a successful cycle")
	}
	if u.failsInARow != 0 {
		t.Fatalf("failsInARow = %d, want 0", u.failsInARow)
	}
}

// After too many consecutive failures the unlocker re-latches: it stops retrying
// (a restart is required) instead of looping forever on a wedged node.
func TestUnlockerRelatchesAfterPersistentFailure(t *testing.T) {
	fake := &fakeUnlockerRPC{pendingErr: errors.New("wedged")}
	u := newTestUnlocker(fake)

	for i := 0; i < maxUnlockFailsInARow; i++ {
		u.runCycle()
	}
	if u.failsInARow != maxUnlockFailsInARow {
		t.Fatalf("failsInARow = %d, want %d", u.failsInARow, maxUnlockFailsInARow)
	}

	// The next cycle is latched: it neither runs nor recovers.
	u.runCycle()
	if !u.halt {
		t.Fatal("must stay halted after the failure threshold")
	}
	if u.failsInARow != maxUnlockFailsInARow {
		t.Fatalf("a latched cycle must not run; failsInARow = %d, want %d", u.failsInARow, maxUnlockFailsInARow)
	}
}
