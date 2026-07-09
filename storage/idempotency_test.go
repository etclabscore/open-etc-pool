package storage

import (
	"math/big"
	"testing"
)

// The unlocker's cycle is safe to retry because a block is credited AND removed
// from the scanned zset in the same atomic transaction: after crediting, a
// re-scan finds nothing, so the credit can't be repeated on a retry. Note the
// immature-balance HIncrBy is itself unconditional — the safety comes from the
// set removal being part of the same EXEC, not from the credit log's HSetNX.
func TestUnlockerCreditIsAtomicWithSetRemoval(t *testing.T) {
	reset()

	r.WriteShare("0xaa", "rig1", []string{"0x1", "0xp1", "0xm1"}, 100, 500, 0)
	r.WriteBlock("0xaa", "rig1", []string{"0x2", "0xp2", "0xm2"}, 100, 5000, 500, 0)

	candidates, _ := r.GetCandidates(1000)
	if len(candidates) != 1 {
		t.Fatalf("need one candidate, got %d", len(candidates))
	}
	block := candidates[0]
	block.Hash = "0xhash"
	block.Reward = big.NewInt(1000000000000000000)
	rewards := map[string]int64{"0xaa": 100}

	// Immature: crediting and removing the candidate are one transaction.
	if err := r.WriteImmatureBlock(block, rewards); err != nil {
		t.Fatalf("WriteImmatureBlock: %v", err)
	}
	if c, _ := r.GetCandidates(1000); len(c) != 0 {
		t.Fatal("candidate must be gone from the scanned set after crediting")
	}
	if got := minerImmature(t, "0xaa"); got != 100 {
		t.Fatalf("immature = %d, want 100", got)
	}
	// A retried cycle re-scans candidates and finds nothing -> no double credit.
	if c, _ := r.GetCandidates(1000); len(c) != 0 {
		t.Fatal("re-scan must be empty (this is what makes a retry safe)")
	}

	// Matured: the same invariant holds for immature -> matured.
	immature, _ := r.GetImmatureBlocks(1000)
	if len(immature) != 1 {
		t.Fatalf("need one immature block, got %d", len(immature))
	}
	imblock := immature[0]
	imblock.Reward = big.NewInt(1000000000000000000)
	if err := r.WriteMaturedBlock(imblock, rewards); err != nil {
		t.Fatalf("WriteMaturedBlock: %v", err)
	}
	if im, _ := r.GetImmatureBlocks(1000); len(im) != 0 {
		t.Fatal("block must be gone from immature after maturing")
	}
	if bal, _ := r.GetBalance("0xaa"); bal != 100 {
		t.Fatalf("balance = %d, want 100", bal)
	}
	if got := minerImmature(t, "0xaa"); got != 0 {
		t.Fatalf("immature after maturing = %d, want 0", got)
	}
}
