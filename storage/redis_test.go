package storage

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestWriteBlockRecordsCandidateAtomically(t *testing.T) {
	reset()
	// A round in progress.
	r.client.HSet(ctx, r.formatKey("shares", "roundCurrent"), "0xaa", 100, "0xbb", 50)
	r.client.HSet(ctx, r.formatKey("stats"), "roundShares", 150)

	params := []string{"0xnonce", "0xpow", "0xmix"}
	exist, err := r.WriteBlock("0xcc", "rig", params, 10, 1000, 42, time.Hour)
	if err != nil || exist {
		t.Fatalf("WriteBlock: exist=%v err=%v", exist, err)
	}

	// The round was renamed with the solver's own share added; roundCurrent gone.
	shares, _ := r.GetRoundShares(42, "0xnonce")
	if shares["0xaa"] != 100 || shares["0xbb"] != 50 || shares["0xcc"] != 10 {
		t.Fatalf("round shares after block = %v", shares)
	}
	if n, _ := r.client.Exists(ctx, r.formatKey("shares", "roundCurrent")).Result(); n != 0 {
		t.Fatal("roundCurrent should have been renamed away")
	}

	// The candidate was recorded atomically with totalShares = the round sum.
	cands, err := r.GetCandidates(100)
	if err != nil {
		t.Fatalf("GetCandidates: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("want 1 candidate, got %d", len(cands))
	}
	c := cands[0]
	if c.Height != 42 || c.Nonce != "0xnonce" || c.Difficulty != 1000 {
		t.Fatalf("candidate mismatch: %+v", c)
	}
	if c.TotalShares != 160 {
		t.Fatalf("candidate TotalShares = %d, want 160 (100+50+10)", c.TotalShares)
	}

	// roundShares cleared and blocksFound incremented.
	if ok, _ := r.client.HExists(ctx, r.formatKey("stats"), "roundShares").Result(); ok {
		t.Fatal("roundShares should have been deleted")
	}
	if bf, _ := r.client.HGet(ctx, r.formatKey("miners", "0xcc"), "blocksFound").Int64(); bf != 1 {
		t.Fatalf("blocksFound = %d, want 1", bf)
	}
}

var r *RedisClient

const prefix = "test"

func TestMain(m *testing.M) {
	r = NewRedisClient(&Config{Endpoint: "127.0.0.1:6379"}, prefix)
	reset()
	c := m.Run()
	reset()
	os.Exit(c)
}

func TestWriteShareCheckExist(t *testing.T) {
	reset()

	exist, _ := r.WriteShare("x", "x", []string{"0x0", "0x0", "0x0"}, 10, 1008, 0)
	if exist {
		t.Error("PoW must not exist")
	}
	exist, _ = r.WriteShare("x", "x", []string{"0x0", "0x1", "0x0"}, 10, 1008, 0)
	if exist {
		t.Error("PoW must not exist")
	}
	exist, _ = r.WriteShare("x", "x", []string{"0x0", "0x0", "0x1"}, 100, 1010, 0)
	if exist {
		t.Error("PoW must not exist")
	}
	exist, _ = r.WriteShare("z", "x", []string{"0x0", "0x0", "0x1"}, 100, 1016, 0)
	if !exist {
		t.Error("PoW must exist")
	}
	exist, _ = r.WriteShare("x", "x", []string{"0x0", "0x0", "0x1"}, 100, 1025, 0)
	if exist {
		t.Error("PoW must not exist")
	}
}

func TestGetPayees(t *testing.T) {
	reset()

	n := 256
	for i := 0; i < n; i++ {
		r.client.HSet(ctx, r.formatKey("miners", strconv.Itoa(i)), "balance", strconv.Itoa(i))
	}

	var payees []string
	payees, _ = r.GetPayees()
	if len(payees) != n {
		t.Error("Must return all payees")
	}
	m := make(map[string]struct{})
	for _, v := range payees {
		m[v] = struct{}{}
	}
	if len(m) != n {
		t.Error("Must be unique list")
	}
}

func TestGetBalance(t *testing.T) {
	reset()

	r.client.HSet(ctx, r.formatKey("miners:x"), "balance", "750")

	v, _ := r.GetBalance("x")
	if v != 750 {
		t.Error("Must return balance")
	}

	v, err := r.GetBalance("z")
	if v != 0 {
		t.Error("Must return 0 if account does not exist")
	}
	if err != nil {
		t.Error("Must not return error if account does not exist")
	}
}

func TestLockPayouts(t *testing.T) {
	reset()

	r.LockPayouts("x", 1000)
	v := r.client.Get(ctx, "test:payments:lock").Val()
	if v != "x:1000" {
		t.Errorf("Invalid lock amount: %v", v)
	}

	err := r.LockPayouts("x", 100)
	if err == nil {
		t.Errorf("Must not overwrite lock")
	}
}

func TestUnlockPayouts(t *testing.T) {
	reset()

	r.client.Set(ctx, r.formatKey("payments:lock"), "x:1000", 0)

	r.UnlockPayouts()
	err := r.client.Get(ctx, r.formatKey("payments:lock")).Err()
	if err != redis.Nil {
		t.Errorf("Must release lock")
	}
}

func TestIsPayoutsLocked(t *testing.T) {
	reset()

	r.LockPayouts("x", 1000)
	if locked, _ := r.IsPayoutsLocked(); !locked {
		t.Errorf("Payouts must be locked")
	}
}

func TestUpdateBalance(t *testing.T) {
	reset()

	r.client.HSet(
		ctx,
		r.formatKey("miners:x"),
		map[string]string{"paid": "50", "balance": "1000"},
	)
	r.client.HSet(
		ctx,
		r.formatKey("finances"),
		map[string]string{"paid": "500", "balance": "10000"},
	)

	amount := int64(250)
	r.UpdateBalance("x", amount)
	result := r.client.HGetAll(ctx, r.formatKey("miners:x")).Val()
	if result["pending"] != "250" {
		t.Error("Must set pending amount")
	}
	if result["balance"] != "750" {
		t.Error("Must deduct balance")
	}
	if result["paid"] != "50" {
		t.Error("Must not touch paid")
	}

	result = r.client.HGetAll(ctx, r.formatKey("finances")).Val()
	if result["pending"] != "250" {
		t.Error("Must set pool pending amount")
	}
	if result["balance"] != "9750" {
		t.Error("Must deduct pool balance")
	}
	if result["paid"] != "500" {
		t.Error("Must not touch pool paid")
	}

	rank := r.client.ZRank(ctx, r.formatKey("payments:pending"), join("x", amount)).Val()
	if rank != 0 {
		t.Error("Must add pending payment")
	}
}

func TestRollbackBalance(t *testing.T) {
	reset()

	r.client.HSet(
		ctx,
		r.formatKey("miners:x"),
		map[string]string{"paid": "100", "balance": "750", "pending": "250"},
	)
	r.client.HSet(
		ctx,
		r.formatKey("finances"),
		map[string]string{"paid": "500", "balance": "10000", "pending": "250"},
	)
	r.client.ZAdd(ctx, r.formatKey("payments:pending"), redis.Z{Score: 1, Member: "xx"})

	amount := int64(250)
	r.RollbackBalance("x", amount)
	result := r.client.HGetAll(ctx, r.formatKey("miners:x")).Val()
	if result["paid"] != "100" {
		t.Error("Must not touch paid")
	}
	if result["balance"] != "1000" {
		t.Error("Must increase balance")
	}
	if result["pending"] != "0" {
		t.Error("Must deduct pending")
	}

	result = r.client.HGetAll(ctx, r.formatKey("finances")).Val()
	if result["paid"] != "500" {
		t.Error("Must not touch pool paid")
	}
	if result["balance"] != "10250" {
		t.Error("Must increase pool balance")
	}
	if result["pending"] != "0" {
		t.Error("Must deduct pool pending")
	}

	err := r.client.ZRank(ctx, r.formatKey("payments:pending"), join("x", amount)).Err()
	if err != redis.Nil {
		t.Errorf("Must remove pending payment")
	}
}

func TestWritePayment(t *testing.T) {
	reset()

	r.client.HSet(
		ctx,
		r.formatKey("miners:x"),
		map[string]string{"paid": "50", "balance": "1000", "pending": "250"},
	)
	r.client.HSet(
		ctx,
		r.formatKey("finances"),
		map[string]string{"paid": "500", "balance": "10000", "pending": "250"},
	)

	amount := int64(250)
	r.WritePayment("x", "0x0", amount)
	result := r.client.HGetAll(ctx, r.formatKey("miners:x")).Val()
	if result["pending"] != "0" {
		t.Error("Must unset pending amount")
	}
	if result["balance"] != "1000" {
		t.Error("Must not touch balance")
	}
	if result["paid"] != "300" {
		t.Error("Must increase paid")
	}

	result = r.client.HGetAll(ctx, r.formatKey("finances")).Val()
	if result["pending"] != "0" {
		t.Error("Must deduct pool pending amount")
	}
	if result["balance"] != "10000" {
		t.Error("Must not touch pool balance")
	}
	if result["paid"] != "750" {
		t.Error("Must increase pool paid")
	}

	err := r.client.Get(ctx, r.formatKey("payments:lock")).Err()
	if err != redis.Nil {
		t.Errorf("Must release lock")
	}

	err = r.client.ZRank(ctx, r.formatKey("payments:pending"), join("x", amount)).Err()
	if err != redis.Nil {
		t.Error("Must remove pending payment")
	}
	err = r.client.ZRank(ctx, r.formatKey("payments:all"), join("0x0", "x", amount)).Err()
	if err == redis.Nil {
		t.Error("Must add payment to set")
	}
	err = r.client.ZRank(ctx, r.formatKey("payments:x"), join("0x0", amount)).Err()
	if err == redis.Nil {
		t.Error("Must add payment to set")
	}
}

func TestGetPendingPayments(t *testing.T) {
	reset()

	r.client.HSet(
		ctx,
		r.formatKey("miners:x"),
		map[string]string{"paid": "100", "balance": "750", "pending": "250"},
	)

	amount := int64(1000)
	r.UpdateBalance("x", amount)
	pending := r.GetPendingPayments()

	if len(pending) != 1 {
		t.Error("Must return pending payment")
	}
	if pending[0].Amount != amount {
		t.Error("Must have corrent amount")
	}
	if pending[0].Address != "x" {
		t.Error("Must have corrent account")
	}
	if pending[0].Timestamp <= 0 {
		t.Error("Must have timestamp")
	}
}

func TestCollectLuckStats(t *testing.T) {
	reset()

	members := []redis.Z{
		redis.Z{Score: 0, Member: "1:0:0x0:0x0:0:100:100:0"},
	}
	r.client.ZAdd(ctx, r.formatKey("blocks:immature"), members...)
	members = []redis.Z{
		redis.Z{Score: 1, Member: "1:0:0x2:0x0:0:50:100:0"},
		redis.Z{Score: 2, Member: "0:1:0x1:0x0:0:100:100:0"},
		redis.Z{Score: 3, Member: "0:0:0x3:0x0:0:200:100:0"},
	}
	r.client.ZAdd(ctx, r.formatKey("blocks:matured"), members...)

	stats, _ := r.CollectLuckStats([]int{1, 2, 5, 10})
	expectedStats := map[string]interface{}{
		"1": map[string]float64{
			"luck": 1, "uncleRate": 1, "orphanRate": 0,
		},
		"2": map[string]float64{
			"luck": 0.75, "uncleRate": 0.5, "orphanRate": 0,
		},
		"4": map[string]float64{
			"luck": 1.125, "uncleRate": 0.5, "orphanRate": 0.25,
		},
	}

	if !reflect.DeepEqual(stats, expectedStats) {
		t.Error("Stats != expected stats")
	}
}

func TestCheckPoWExistLowHeightDedup(t *testing.T) {
	reset()
	params := []string{"0xn", "0xp", "0xm"}
	// height 3 (< 8): the backlog sweep must be skipped so a uint64 underflow
	// doesn't prune the dedup entry, and the duplicate is still detected.
	if exist, err := r.checkPoWExist(3, params); err != nil || exist {
		t.Fatalf("first submit at height 3: exist=%v err=%v", exist, err)
	}
	if exist, err := r.checkPoWExist(3, params); err != nil || !exist {
		t.Fatalf("duplicate submit at height 3 must be detected: exist=%v err=%v", exist, err)
	}
}

func reset() {
	keys := r.client.Keys(ctx, r.prefix+":*").Val()
	for _, k := range keys {
		r.client.Del(ctx, k)
	}
}

func TestLockPayoutsContention(t *testing.T) {
	reset()

	if err := r.LockPayouts("0xaa", 5); err != nil {
		t.Fatalf("first lock should succeed: %v", err)
	}
	locked, err := r.IsPayoutsLocked()
	if err != nil || !locked {
		t.Fatalf("payouts should be locked: locked=%v err=%v", locked, err)
	}
	// A second lock while the first is held is genuine contention, not a
	// backend failure.
	if err := r.LockPayouts("0xbb", 7); err == nil {
		t.Fatal("second lock should fail while the first is held")
	}
	if err := r.UnlockPayouts(); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if locked, _ := r.IsPayoutsLocked(); locked {
		t.Fatal("payouts should be unlocked after UnlockPayouts")
	}
}

func TestLockPayoutsSurfacesBackendError(t *testing.T) {
	// Pointed at a closed port, LockPayouts must return the real connection
	// error, not the "Unable to acquire lock" contention message that swallowing
	// the SetNX error with .Val() would have produced.
	down := NewRedisClient(&Config{Endpoint: "127.0.0.1:1"}, prefix)
	err := down.LockPayouts("0xaa", 5)
	if err == nil {
		t.Fatal("expected an error when the backend is unreachable")
	}
	if strings.Contains(err.Error(), "Unable to acquire lock") {
		t.Fatalf("backend error masked as lock contention: %v", err)
	}
}
