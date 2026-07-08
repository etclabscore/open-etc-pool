package storage

import (
	"math/big"
	"testing"
	"time"
)

// Characterization tests for the block lifecycle and stats-collection paths.
//
// These use ONLY the public RedisClient API (no r.client / ctx / redis.*), so
// the exact same file compiles and runs against both the old gopkg.in/redis.v3
// backend and the new redis/go-redis/v9 backend. Running them before AND after
// the client migration proves the swap preserves behavior. They cover the
// highest-risk methods the previous suite left untested: WriteBlock (cmds[10]),
// the immature -> matured / orphan credit flow (the Watch rewrite), CollectStats
// and GetMinerStats (positional pipeline-result casts).

func TestWriteBlockCandidate(t *testing.T) {
	reset()

	// Accumulate round shares from two miners.
	r.WriteShare("0xaa", "rig1", []string{"0x1", "0xp1", "0xm1"}, 50, 100, 0)
	r.WriteShare("0xbb", "rig1", []string{"0x2", "0xp2", "0xm2"}, 30, 100, 0)

	// Block found (its own share counts too: +100 for 0xaa). params[0] is the
	// nonce used to key the round.
	exist, err := r.WriteBlock("0xaa", "rig2", []string{"0x3", "0xp3", "0xm3"}, 100, 12345, 100, 0)
	if err != nil {
		t.Fatalf("WriteBlock error: %v", err)
	}
	if exist {
		t.Error("Block PoW must not exist")
	}

	candidates, err := r.GetCandidates(1000)
	if err != nil {
		t.Fatalf("GetCandidates error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("Must record exactly one candidate, got %d", len(candidates))
	}
	c := candidates[0]
	if c.Height != 100 {
		t.Errorf("Candidate height = %d, want 100", c.Height)
	}
	if c.Nonce != "0x3" {
		t.Errorf("Candidate nonce = %q, want 0x3", c.Nonce)
	}
	// TotalShares snapshot = 50 (0xaa share) + 100 (0xaa block share) + 30 (0xbb) = 180
	if c.TotalShares != 180 {
		t.Errorf("Candidate totalShares = %d, want 180", c.TotalShares)
	}

	// roundCurrent must have been renamed to round<height>:<nonce>.
	shares, err := r.GetRoundShares(100, "0x3")
	if err != nil {
		t.Fatalf("GetRoundShares error: %v", err)
	}
	if shares["0xaa"] != 150 {
		t.Errorf("Round shares[0xaa] = %d, want 150", shares["0xaa"])
	}
	if shares["0xbb"] != 30 {
		t.Errorf("Round shares[0xbb] = %d, want 30", shares["0xbb"])
	}
}

func minerImmature(t *testing.T, login string) int64 {
	t.Helper()
	stats, err := r.GetMinerStats(login, 10)
	if err != nil {
		t.Fatalf("GetMinerStats error: %v", err)
	}
	m := stats["stats"].(map[string]interface{})
	v, ok := m["immature"]
	if !ok {
		return 0
	}
	return v.(int64)
}

func TestBlockMaturation(t *testing.T) {
	reset()

	r.WriteShare("0xaa", "rig1", []string{"0x1", "0xp1", "0xm1"}, 60, 200, 0)
	r.WriteBlock("0xaa", "rig1", []string{"0x2", "0xp2", "0xm2"}, 40, 5000, 200, 0)

	candidates, _ := r.GetCandidates(1000)
	if len(candidates) != 1 {
		t.Fatalf("Must have one candidate")
	}
	block := candidates[0]
	// The unlocker sets these on the matched block before crediting.
	block.Hash = "0xblockhash"
	block.Reward = big.NewInt(1000000000000000000)
	roundRewards := map[string]int64{"0xaa": 100}

	if err := r.WriteImmatureBlock(block, roundRewards); err != nil {
		t.Fatalf("WriteImmatureBlock error: %v", err)
	}
	if got := minerImmature(t, "0xaa"); got != 100 {
		t.Errorf("immature = %d, want 100", got)
	}
	if c, _ := r.GetCandidates(1000); len(c) != 0 {
		t.Errorf("candidate must be removed after immature write, got %d", len(c))
	}

	immature, _ := r.GetImmatureBlocks(1000)
	if len(immature) != 1 {
		t.Fatalf("Must have one immature block")
	}
	imblock := immature[0]
	imblock.Reward = big.NewInt(1000000000000000000)

	// Maturation exercises the Watch (optimistic-locking) rewrite.
	if err := r.WriteMaturedBlock(imblock, roundRewards); err != nil {
		t.Fatalf("WriteMaturedBlock error: %v", err)
	}
	if bal, _ := r.GetBalance("0xaa"); bal != 100 {
		t.Errorf("balance = %d, want 100", bal)
	}
	if got := minerImmature(t, "0xaa"); got != 0 {
		t.Errorf("immature after maturation = %d, want 0", got)
	}
}

func TestBlockOrphan(t *testing.T) {
	reset()

	r.WriteShare("0xcc", "rig1", []string{"0x1", "0xp1", "0xm1"}, 60, 300, 0)
	r.WriteBlock("0xcc", "rig1", []string{"0x2", "0xp2", "0xm2"}, 40, 5000, 300, 0)

	candidates, _ := r.GetCandidates(1000)
	block := candidates[0]
	block.Hash = "0xorphanhash"
	block.Reward = big.NewInt(1000000000000000000)

	if err := r.WriteImmatureBlock(block, map[string]int64{"0xcc": 80}); err != nil {
		t.Fatalf("WriteImmatureBlock error: %v", err)
	}
	if got := minerImmature(t, "0xcc"); got != 80 {
		t.Fatalf("immature = %d, want 80", got)
	}

	immature, _ := r.GetImmatureBlocks(1000)
	imblock := immature[0]
	imblock.Reward = big.NewInt(1000000000000000000)

	// Orphaning also exercises the Watch rewrite; it must revert immature and
	// credit no balance.
	if err := r.WriteOrphan(imblock); err != nil {
		t.Fatalf("WriteOrphan error: %v", err)
	}
	if got := minerImmature(t, "0xcc"); got != 0 {
		t.Errorf("immature after orphan = %d, want 0", got)
	}
	if bal, _ := r.GetBalance("0xcc"); bal != 0 {
		t.Errorf("balance after orphan = %d, want 0", bal)
	}
}

func TestCollectStatsShape(t *testing.T) {
	reset()

	r.WriteShare("0xaa", "rig1", []string{"0x1", "0xp1", "0xm1"}, 100, 400, time.Hour)
	r.WriteBlock("0xaa", "rig1", []string{"0x2", "0xp2", "0xm2"}, 100, 5000, 400, time.Hour)

	stats, err := r.CollectStats(time.Hour, 50, 50)
	if err != nil {
		t.Fatalf("CollectStats error: %v", err)
	}
	// All positional casts must succeed (a wrong index/type would panic) and
	// every documented key must be present.
	for _, k := range []string{
		"stats", "candidates", "candidatesTotal", "immature", "immatureTotal",
		"matured", "maturedTotal", "payments", "paymentsTotal", "miners",
		"minersTotal", "hashrate",
	} {
		if _, ok := stats[k]; !ok {
			t.Errorf("CollectStats missing key %q", k)
		}
	}
	if stats["candidatesTotal"].(int64) != 1 {
		t.Errorf("candidatesTotal = %v, want 1", stats["candidatesTotal"])
	}
	if cands := stats["candidates"].([]*BlockData); len(cands) != 1 {
		t.Errorf("candidates len = %d, want 1", len(cands))
	}
}

func TestGetMinerStatsShape(t *testing.T) {
	reset()

	r.WriteShare("0xaa", "rig1", []string{"0x1", "0xp1", "0xm1"}, 100, 500, time.Hour)

	stats, err := r.GetMinerStats("0xaa", 30)
	if err != nil {
		t.Fatalf("GetMinerStats error: %v", err)
	}
	for _, k := range []string{"stats", "payments", "paymentsTotal", "roundShares"} {
		if _, ok := stats[k]; !ok {
			t.Errorf("GetMinerStats missing key %q", k)
		}
	}
	if stats["roundShares"].(int64) != 100 {
		t.Errorf("roundShares = %v, want 100", stats["roundShares"])
	}
}
