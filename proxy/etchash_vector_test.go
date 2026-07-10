package proxy

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// These vectors validate go-etchash's PoW verification — the exact library and
// code path proxy.processShare relies on — against blocks the Ethereum Classic
// chain actually accepted, i.e. an external source of truth rather than our own
// Compute. The two blocks straddle the ECIP-1099 activation (11,700,000) so both
// DAG-epoch regimes (30000- and 60000-length) are exercised.
//
// Etchash hashes the header's *seal hash* (RLP of the header without nonce and
// mixHash), which no block explorer serves directly; sealHash recomputes it from
// the full header. Verify only returns true if compute(sealHash, nonce)
// reproduces the block's real mixHash, so a passing test simultaneously proves
// (a) our seal hash matches the one the chain used and (b) the Etchash PoW result
// clears the block's difficulty.
//
// Header fields were fetched from a public ETC node (eth_getBlockByNumber).

type etcHeader struct {
	number      uint64
	difficulty  string
	nonce       string
	mixHash     string
	parentHash  string
	sha3Uncles  string
	miner       string
	stateRoot   string
	txRoot      string
	receiptRoot string
	gasLimit    string
	gasUsed     string
	timestamp   string
	extraData   string
	logsBloom   string // "" means an all-zero bloom
}

func hexBig(s string) *big.Int {
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !ok {
		panic("bad hex big: " + s)
	}
	return n
}

// sealHash reproduces the Etchash proof-of-work input: Keccak-256 of the
// RLP-encoded header without its nonce and mixHash. This mirrors go-ethereum's
// ethash.SealHash for pre-London headers (the 13-field form); Ethereum Classic
// never adopted EIP-1559, so its headers carry no base fee.
func (v etcHeader) sealHash() common.Hash {
	bloom := make([]byte, 256)
	if v.logsBloom != "" {
		bloom = hexutil.MustDecode(v.logsBloom)
	}
	hasher := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hasher, []interface{}{
		common.HexToHash(v.parentHash),
		common.HexToHash(v.sha3Uncles),
		common.HexToAddress(v.miner),
		common.HexToHash(v.stateRoot),
		common.HexToHash(v.txRoot),
		common.HexToHash(v.receiptRoot),
		bloom,
		hexBig(v.difficulty),
		new(big.Int).SetUint64(v.number),
		hexutil.MustDecodeUint64(v.gasLimit),
		hexutil.MustDecodeUint64(v.gasUsed),
		hexutil.MustDecodeUint64(v.timestamp),
		hexutil.MustDecode(v.extraData),
	})
	var hash common.Hash
	hasher.Sum(hash[:0])
	return hash
}

var etcVectors = []etcHeader{
	{ // block 10,000,000 — pre-ECIP-1099 (30000-length epoch)
		number:      10000000,
		difficulty:  "0x8d51aa3e06d7",
		nonce:       "0x2e56400011aa68bb",
		mixHash:     "0xeeecef23e81ea2b635fe4ddb1b0a73f672185442c580d7ef3c069269566b9fe6",
		parentHash:  "0xc30ce5be0729bbba8fdfad0a7e63e144661533e261b8c46fe8c17f8ae32cbbc5",
		sha3Uncles:  "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
		miner:       "0x0073cf1b9230cf3ee8cab1971b8dbef21ea7b595",
		stateRoot:   "0x72295b0f3a32caca037dc0b2941517c42f58cc322fd59d93fb9f4343690062fe",
		txRoot:      "0x222c814f9fa5fe63899dd415623eea2a31df844743fbda32339b788fad74ea5b",
		receiptRoot: "0x5a192cfacfa828e351d620219f8b164bc450c03a3ee437e136de609250d9d05c",
		gasLimit:    "0x79f39e",
		gasUsed:     "0x6b3220",
		timestamp:   "0x5e7068d0",
		extraData:   "0x457468657265756d436c617373696350504c4e532f326d696e657273",
	},
	{ // block 15,000,000 — post-ECIP-1099 (60000-length epoch)
		number:      15000000,
		difficulty:  "0x144350f51ecab",
		nonce:       "0x51722f9f5fe9570d",
		mixHash:     "0xd42078e487ac321317e73b98cf8f6f70e5c30bbc38c1cf8d11fc03add955a096",
		parentHash:  "0xebf1b5d82c42d8fa8dcce8cb2a3a24cc9247a2781bed92e692db93a1061cc8e2",
		sha3Uncles:  "0x04f61d1bb93de95612070cddb43a70dcf8170395029266b9a85e300a30ace389",
		miner:       "0x8ccfe15255cddcd20fc667fc508936c74e91e5c1",
		stateRoot:   "0x050bc26b273da87b05812dc37c9dbc22399fa012d3833a7b6e0addbb3debbcb7",
		txRoot:      "0xc068e810273bb69f4bf9a87de71a9ca07b51df4db2a139155bd7bdc54e3368b4",
		receiptRoot: "0x1cb808c3d7074667a9b8014a6d1c2b8e77ab8a4f4ea4fc21aa82ff3cbe874a30",
		gasLimit:    "0x7a1200",
		gasUsed:     "0x260bb",
		timestamp:   "0x62668324",
		extraData:   "0x43727578706f6f6c205050532f7573",
		logsBloom:   "0x00000000000000000000000000000100004080000000000000000000000000000000002000000000000000000000000000000000000004000000000041000000000000000000000000000000000000000000000000000000001000000020200000000040020000000000000000800820002000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000200000000000080000000000000000000000000000000000000000000000100002000000020000000000000020000000000000000000000000000000000000000000040000000000000000000000",
	},
}

func TestVerifyRealETCBlocks(t *testing.T) {
	verifier := getHasher("classic")
	if verifier == nil {
		t.Fatal("getHasher returned nil")
	}

	for _, v := range etcVectors {
		block := Block{
			number:      v.number,
			hashNoNonce: v.sealHash(),
			difficulty:  hexBig(v.difficulty),
			nonce:       hexutil.MustDecodeUint64(v.nonce),
			mixDigest:   common.HexToHash(v.mixHash),
		}

		if !verifier.Verify(block) {
			t.Fatalf("block %d: real ETC block failed Etchash verification (seal=%s)", v.number, block.hashNoNonce.Hex())
		}

		// Flipping the nonce must break verification — proves Verify is really
		// checking the PoW, not trivially returning true.
		tampered := block
		tampered.nonce = block.nonce ^ 1
		if verifier.Verify(tampered) {
			t.Fatalf("block %d: verification passed with a tampered nonce", v.number)
		}
	}
}
