package payouts

import (
	"math"
	"math/big"
	"os"
	"testing"

	"github.com/etclabscore/open-etc-pool/rpc"
	"github.com/etclabscore/open-etc-pool/storage"
	"github.com/etclabscore/open-etc-pool/util"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCalculateRewards(t *testing.T) {
	blockReward, _ := new(big.Rat).SetString("5000000000000000000")
	shares := map[string]int64{"0x0": 1000000, "0x1": 20000, "0x2": 5000, "0x3": 10, "0x4": 1}
	expectedRewards := map[string]int64{"0x0": 4877996431, "0x1": 97559929, "0x2": 24389982, "0x3": 48780, "0x4": 4878}
	totalShares := int64(1025011)

	rewards, err := calculateRewardsForShares(shares, totalShares, blockReward)
	if err != nil {
		t.Fatalf("calculateRewardsForShares: %v", err)
	}
	expectedTotalAmount := int64(5000000000)

	totalAmount := int64(0)
	for login, amount := range rewards {
		totalAmount += amount

		if expectedRewards[login] != amount {
			t.Errorf("Amount for %v must be equal to %v vs %v", login, expectedRewards[login], amount)
		}
	}
	if totalAmount != expectedTotalAmount {
		t.Errorf("Total reward must be equal to block reward in Shannon: %v vs %v", expectedTotalAmount, totalAmount)
	}
}

func TestChargeFee(t *testing.T) {
	orig, _ := new(big.Rat).SetString("5000000000000000000")
	value, _ := new(big.Rat).SetString("5000000000000000000")
	expectedNewValue, _ := new(big.Rat).SetString("3750000000000000000")
	expectedFee, _ := new(big.Rat).SetString("1250000000000000000")
	newValue, fee := chargeFee(orig, 25.0)

	if orig.Cmp(value) != 0 {
		t.Error("Must not change original value")
	}
	if newValue.Cmp(expectedNewValue) != 0 {
		t.Error("Must charge and deduct correct fee")
	}
	if fee.Cmp(expectedFee) != 0 {
		t.Error("Must charge fee")
	}
}

func TestWeiToShannonInt64(t *testing.T) {
	wei, _ := new(big.Rat).SetString("1000000000000000000")
	origWei, _ := new(big.Rat).SetString("1000000000000000000")
	shannon := int64(1000000000)

	got, err := weiToShannonInt64(wei)
	if err != nil {
		t.Fatalf("weiToShannonInt64: %v", err)
	}
	if got != shannon {
		t.Error("Must convert to Shannon")
	}
	if wei.Cmp(origWei) != 0 {
		t.Error("Must charge original value")
	}
}

func TestWeiToShannonInt64Overflow(t *testing.T) {
	// (MaxInt64 + 1) Shannon, expressed in Wei, cannot fit int64 Shannon.
	overflowShannon := new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
	wei := new(big.Rat).SetInt(new(big.Int).Mul(overflowShannon, util.Shannon))

	if _, err := weiToShannonInt64(wei); err == nil {
		t.Fatal("expected an overflow error for a reward exceeding int64 Shannon")
	}
}

func TestCalculateRewardsForSharesOverflow(t *testing.T) {
	// A block reward so large a single share's cut overflows int64 Shannon must
	// surface an error rather than silently clamping a miner's balance.
	overflowShannon := new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
	reward := new(big.Rat).SetInt(new(big.Int).Mul(overflowShannon, util.Shannon))

	if _, err := calculateRewardsForShares(map[string]int64{"0xaa": 1}, 1, reward); err == nil {
		t.Fatal("expected an overflow error from calculateRewardsForShares")
	}
}

func TestGetBlockEra(t *testing.T) {
	blockNum := big.NewInt(11700000)
	eraLength := big.NewInt(5000000)
	era := GetBlockEra(blockNum, eraLength)
	if era.Cmp(big.NewInt(2)) != 0 {
		t.Error("Should return Era 2", "era", era)
	}
	// handle negative blockNum
	blockNum = big.NewInt(-50000)
	era = GetBlockEra(blockNum, eraLength)
	if era.Cmp(big.NewInt(0)) != 0 {
		t.Error("Should return Era 0", "era", era)
	}
	// handle negative blockNum
	blockNum = big.NewInt(5000001)
	era = GetBlockEra(blockNum, eraLength)
	if era.Cmp(big.NewInt(1)) != 0 {
		t.Error("Should return Era 1", "era", era)
	}
}

func TestGetBlockWinnerRewardByEra(t *testing.T) {
	baseReward := big.NewInt(5000000000000000000)
	era := big.NewInt(0)
	blockReward := GetBlockWinnerRewardByEra(era, baseReward)
	if blockReward.Cmp(big.NewInt(5000000000000000000)) != 0 {
		t.Error("Should return blockReward 5000000000000000000", "reward", blockReward)
	}
	era = big.NewInt(1)
	blockReward = GetBlockWinnerRewardByEra(era, baseReward)
	if blockReward.Cmp(big.NewInt(4000000000000000000)) != 0 {
		t.Error("Should return blockReward 4000000000000000000", "reward", blockReward)
	}
	era = big.NewInt(2)
	blockReward = GetBlockWinnerRewardByEra(era, baseReward)
	if blockReward.Cmp(big.NewInt(3200000000000000000)) != 0 {
		t.Error("Should return blockReward 3200000000000000000", "reward", blockReward)
	}
	era = big.NewInt(3)
	blockReward = GetBlockWinnerRewardByEra(era, baseReward)
	if blockReward.Cmp(big.NewInt(2560000000000000000)) != 0 {
		t.Error("Should return blockReward 2560000000000000000", "reward", blockReward)
	}
	era = big.NewInt(4)
	blockReward = GetBlockWinnerRewardByEra(era, baseReward)
	if blockReward.Cmp(big.NewInt(2048000000000000000)) != 0 {
		t.Error("Should return blockReward 2048000000000000000", "reward", blockReward)
	}
}

func TestGetRewardForUncle(t *testing.T) {
	baseReward := big.NewInt(4000000000000000000)
	uncleReward := getRewardForUncle(baseReward)
	if uncleReward.Cmp(big.NewInt(125000000000000000)) != 0 {
		t.Error("Should return uncleReward 125000000000000000", "reward", uncleReward)
	}
	baseReward = big.NewInt(3200000000000000000)
	uncleReward = getRewardForUncle(baseReward)
	if uncleReward.Cmp(big.NewInt(100000000000000000)) != 0 {
		t.Error("Should return uncleReward 100000000000000000", "reward", uncleReward)
	}
	baseReward = big.NewInt(2560000000000000000)
	uncleReward = getRewardForUncle(baseReward)
	if uncleReward.Cmp(big.NewInt(80000000000000000)) != 0 {
		t.Error("Should return uncleReward 80000000000000000", "reward", uncleReward)
	}
	baseReward = big.NewInt(2048000000000000000)
	uncleReward = getRewardForUncle(baseReward)
	if uncleReward.Cmp(big.NewInt(64000000000000000)) != 0 {
		t.Error("Should return uncleReward 64000000000000000", "reward", uncleReward)
	}
}

func TestMatchCandidate(t *testing.T) {
	gethBlock := &rpc.GetBlockReply{Hash: "0x12345A", Nonce: "0x1A"}
	parityBlock := &rpc.GetBlockReply{Hash: "0x12345A", SealFields: []string{"0x0A", "0x1A"}}
	candidate := &storage.BlockData{Nonce: "0x1a"}
	orphan := &storage.BlockData{Nonce: "0x1abc"}

	if !matchCandidate(gethBlock, candidate) {
		t.Error("Must match with nonce")
	}
	if !matchCandidate(parityBlock, candidate) {
		t.Error("Must match with seal fields")
	}
	if matchCandidate(gethBlock, orphan) {
		t.Error("Must not match with orphan with nonce")
	}
	if matchCandidate(parityBlock, orphan) {
		t.Error("Must not match orphan with seal fields")
	}

	block := &rpc.GetBlockReply{Hash: "0x12345A"}
	immature := &storage.BlockData{Hash: "0x12345a", Nonce: "0x0"}
	if !matchCandidate(block, immature) {
		t.Error("Must match with hash")
	}
}
