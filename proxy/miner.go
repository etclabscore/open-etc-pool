package proxy

import (
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/etclabscore/go-etchash"
	"github.com/ethereum/go-ethereum/common"
)

var ecip1099FBlockClassic uint64 = 11700000 // classic mainnet
var ecip1099FBlockMordor uint64 = 2520000   // mordor

var (
	hasher     *etchash.Etchash
	hasherOnce sync.Once
)

// getHasher builds the Etchash verifier once, safely under concurrent share
// submissions (this was previously an unsynchronized nil-check + assignment on
// a package global — a data race).
func getHasher(network string) *etchash.Etchash {
	hasherOnce.Do(func() {
		switch network {
		case "classic":
			hasher = etchash.New(&ecip1099FBlockClassic, nil)
		case "mordor":
			hasher = etchash.New(&ecip1099FBlockMordor, nil)
		default:
			log.Printf("Unknown network configuration %s", network)
		}
	})
	return hasher
}

func (s *ProxyServer) processShare(login, id, ip string, t *BlockTemplate, params []string) (bool, bool) {
	verifier := getHasher(s.config.Network)
	if verifier == nil {
		return false, false
	}
	nonceHex := params[0]
	hashNoNonce := params[1]
	mixDigest := params[2]
	nonce, _ := strconv.ParseUint(strings.Replace(nonceHex, "0x", "", -1), 16, 64)
	shareDiff := s.config.Proxy.Difficulty

	h, ok := t.headers[hashNoNonce]
	if !ok {
		log.Printf("Stale share from %v@%v", login, ip)
		return false, false
	}

	share := Block{
		number:      h.height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  big.NewInt(shareDiff),
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	block := Block{
		number:      h.height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  h.diff,
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	if !verifier.Verify(share) {
		return false, false
	}

	if verifier.Verify(block) {
		ok, err := s.rpc().SubmitBlock(params)
		if err != nil {
			log.Printf("Block submission failure at height %v for %v: %v", h.height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", h.height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params, shareDiff, h.diff.Int64(), h.height, s.hashrateExpiration)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
				log.Printf("Inserted block %v to backend", h.height)
			}
			log.Printf("Block found by miner %v@%v at height %d", login, ip, h.height)
		}
	} else {
		exist, err := s.backend.WriteShare(login, id, params, shareDiff, h.height, s.hashrateExpiration)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
