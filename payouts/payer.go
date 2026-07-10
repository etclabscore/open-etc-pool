package payouts

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/etclabscore/open-etc-pool/rpc"
	"github.com/etclabscore/open-etc-pool/storage"
	"github.com/etclabscore/open-etc-pool/util"
)

const txCheckInterval = 5 * time.Second

type PayoutsConfig struct {
	Enabled      bool   `json:"enabled"`
	RequirePeers int64  `json:"requirePeers"`
	Interval     string `json:"interval"`
	Daemon       string `json:"daemon"`
	Timeout      string `json:"timeout"`
	Address      string `json:"address"`
	Gas          string `json:"gas"`
	GasPrice     string `json:"gasPrice"`
	AutoGas      bool   `json:"autoGas"`
	// In Shannon
	Threshold int64 `json:"threshold"`
	BgSave    bool  `json:"bgsave"`
}

func (self PayoutsConfig) GasHex() string {
	x := util.String2Big(self.Gas)
	return hexutil.EncodeBig(x)
}

func (self PayoutsConfig) GasPriceHex() string {
	x := util.String2Big(self.GasPrice)
	return hexutil.EncodeBig(x)
}

type PayoutsProcessor struct {
	config   *PayoutsConfig
	backend  *storage.RedisClient
	rpc      payerRPC
	halt     bool
	lastFail error
}

// payerRPC is the subset of *rpc.RPCClient the payouts processor uses, as an
// interface so tests can inject failures.
type payerRPC interface {
	GetBalance(address string) (*big.Int, error)
	Sign(from string, s string) (string, error)
	GetPeerCount() (int64, error)
	SendTransaction(from, to, gas, gasPrice, value string, autoGas bool) (string, error)
	GetTxReceipt(hash string) (*rpc.TxReceipt, error)
}

func NewPayoutsProcessor(cfg *PayoutsConfig, backend *storage.RedisClient) *PayoutsProcessor {
	u := &PayoutsProcessor{config: cfg, backend: backend}
	u.rpc = rpc.NewRPCClient("PayoutsProcessor", cfg.Daemon, cfg.Timeout)
	return u
}

func (u *PayoutsProcessor) Start(ctx context.Context) {
	log.Println("Starting payouts")

	if u.mustResolvePayout() {
		log.Println("Running with env RESOLVE_PAYOUT=1, now trying to resolve locked payouts")
		u.resolvePayouts()
		log.Println("Now you have to restart payouts module with RESOLVE_PAYOUT=0 for normal run")
		return
	}

	intv := util.MustParseDuration(u.config.Interval)
	timer := time.NewTimer(intv)
	log.Printf("Set payouts interval to %v", intv)

	payments := u.backend.GetPendingPayments()
	if len(payments) > 0 {
		log.Printf("Previous payout failed, you have to resolve it. List of failed payments:\n %v",
			formatPendingPayments(payments))
		return
	}

	locked, err := u.backend.IsPayoutsLocked()
	if err != nil {
		log.Println("Unable to start payouts:", err)
		return
	}
	if locked {
		log.Println("Unable to start payouts because they are locked")
		return
	}

	// Immediately process payouts after start
	u.process()
	timer.Reset(intv)

	for {
		select {
		case <-timer.C:
			u.process()
			timer.Reset(intv)
		case <-ctx.Done():
			log.Println("Stopping payouts")
			timer.Stop()
			return
		}
	}
}

func (u *PayoutsProcessor) process() {
	if u.halt {
		log.Println("Payments suspended due to last critical error:", u.lastFail)
		return
	}
	mustPay := 0
	minersPaid := 0
	totalAmount := big.NewInt(0)
	payees, err := u.backend.GetPayees()
	if err != nil {
		log.Println("Error while retrieving payees from backend:", err)
		return
	}

	for _, login := range payees {
		amount, _ := u.backend.GetBalance(login)
		amountInShannon := big.NewInt(amount)

		// Shannon^2 = Wei
		amountInWei := new(big.Int).Mul(amountInShannon, util.Shannon)

		if !u.reachedThreshold(amountInShannon) {
			continue
		}
		mustPay++

		// Require active peers before processing
		if !u.checkPeers() {
			break
		}
		// Require unlocked account
		if !u.isUnlockedAccount() {
			break
		}

		// Check if we have enough funds. These two checks are before any state
		// mutation, so a transient failure retries next cycle instead of
		// permanently halting payouts.
		poolBalance, err := u.rpc.GetBalance(u.config.Address)
		if err != nil {
			log.Printf("Unable to get pool balance from node, retrying next cycle: %v", err)
			break
		}
		if poolBalance.Cmp(amountInWei) < 0 {
			log.Printf("Not enough pool balance for payment, need %s Wei, pool has %s Wei; retrying next cycle",
				amountInWei.String(), poolBalance.String())
			break
		}

		// Lock payments for current payout
		err = u.backend.LockPayouts(login, amount)
		if err != nil {
			log.Printf("Failed to lock payment for %s: %v", login, err)
			u.halt = true
			u.lastFail = err
			break
		}
		log.Printf("Locked payment for %s, %v Shannon", login, amount)

		// Debit miner's balance and update stats
		err = u.backend.UpdateBalance(login, amount)
		if err != nil {
			log.Printf("Failed to update balance for %s, %v Shannon: %v", login, amount, err)
			u.halt = true
			u.lastFail = err
			break
		}

		value := hexutil.EncodeBig(amountInWei)
		txHash, err := u.rpc.SendTransaction(u.config.Address, login, u.config.GasHex(), u.config.GasPriceHex(), value, u.config.AutoGas)
		if err != nil {
			log.Printf("Failed to send payment to %s, %v Shannon: %v. Check outgoing tx for %s in block explorer and docs/PAYOUTS.md",
				login, amount, err, login)
			u.halt = true
			u.lastFail = err
			break
		}

		// Record the broadcast tx hash before WritePayment, so that a crash here
		// leaves enough state for resolvePayouts to recognise this payout as
		// already sent and not credit the balance back (which would double-pay).
		err = u.backend.SetPendingPaymentTx(login, amount, txHash)
		if err != nil {
			log.Printf("Failed to record pending tx %s for %s: %v", txHash, login, err)
			u.halt = true
			u.lastFail = err
			break
		}

		// Log transaction hash
		err = u.backend.WritePayment(login, txHash, amount)
		if err != nil {
			log.Printf("Failed to log payment data for %s, %v Shannon, tx: %s: %v", login, amount, txHash, err)
			u.halt = true
			u.lastFail = err
			break
		}

		minersPaid++
		totalAmount.Add(totalAmount, big.NewInt(amount))
		log.Printf("Paid %v Shannon to %v, TxHash: %v", amount, login, txHash)

		// Wait for TX confirmation before further payouts
		for {
			log.Printf("Waiting for tx confirmation: %v", txHash)
			time.Sleep(txCheckInterval)
			receipt, err := u.rpc.GetTxReceipt(txHash)
			if err != nil {
				log.Printf("Failed to get tx receipt for %v: %v", txHash, err)
				continue
			}
			// Tx has been mined
			if receipt != nil && receipt.Confirmed() {
				if receipt.Successful() {
					log.Printf("Payout tx successful for %s: %s", login, txHash)
				} else {
					log.Printf("Payout tx failed for %s: %s. Address contract throws on incoming tx.", login, txHash)
				}
				break
			}
		}
	}

	if mustPay > 0 {
		log.Printf("Paid total %v Shannon to %v of %v payees", totalAmount, minersPaid, mustPay)
	} else {
		log.Println("No payees that have reached payout threshold")
	}

	// Save redis state to disk
	if minersPaid > 0 && u.config.BgSave {
		u.bgSave()
	}
}

func (self PayoutsProcessor) isUnlockedAccount() bool {
	_, err := self.rpc.Sign(self.config.Address, "0x0")
	if err != nil {
		log.Println("Unable to process payouts:", err)
		return false
	}
	return true
}

func (self PayoutsProcessor) checkPeers() bool {
	n, err := self.rpc.GetPeerCount()
	if err != nil {
		log.Println("Unable to start payouts, failed to retrieve number of peers from node:", err)
		return false
	}
	if n < self.config.RequirePeers {
		log.Println("Unable to start payouts, number of peers on a node is less than required", self.config.RequirePeers)
		return false
	}
	return true
}

func (self PayoutsProcessor) reachedThreshold(amount *big.Int) bool {
	return big.NewInt(self.config.Threshold).Cmp(amount) < 0
}

func formatPendingPayments(list []*storage.PendingPayment) string {
	var s string
	for _, v := range list {
		s += fmt.Sprintf("\tAddress: %s, Amount: %v Shannon, %v\n", v.Address, v.Amount, time.Unix(v.Timestamp, 0))
	}
	return s
}

func (self PayoutsProcessor) bgSave() {
	result, err := self.backend.BgSave()
	if err != nil {
		log.Println("Failed to perform BGSAVE on backend:", err)
		return
	}
	log.Println("Saving backend state to disk:", result)
}

func (self PayoutsProcessor) resolvePayouts() {
	payments := self.backend.GetPendingPayments()

	if len(payments) > 0 {
		log.Printf("Resolving %v pending payment(s):\n%s", len(payments), formatPendingPayments(payments))

		for _, v := range payments {
			txHash, err := self.backend.GetPendingPaymentTx(v.Address, v.Amount)
			if err != nil {
				log.Printf("Failed to read pending tx for %s, leaving it for manual resolution: %v", v.Address, err)
				return
			}

			// No tx was ever broadcast (crash before or during send), so the
			// payout never left the pool and the balance can be credited back.
			if txHash == "" {
				if err := self.backend.RollbackBalance(v.Address, v.Amount); err != nil {
					log.Printf("Failed to credit %v Shannon back to %s: %v", v.Amount, v.Address, err)
					return
				}
				log.Printf("Credited %v Shannon back to %s (no payout tx was broadcast)", v.Amount, v.Address)
				continue
			}

			// A payout tx was broadcast. Only credit back if it provably reverted
			// on-chain; when it succeeded, is still pending, or can't be checked,
			// treat it as paid so an already-sent payout is never double-paid.
			receipt, err := self.rpc.GetTxReceipt(txHash)
			if err == nil && receipt != nil && receipt.Confirmed() && !receipt.Successful() {
				if err := self.backend.RollbackBalance(v.Address, v.Amount); err != nil {
					log.Printf("Failed to credit %v Shannon back to %s: %v", v.Amount, v.Address, err)
					return
				}
				log.Printf("Credited %v Shannon back to %s (payout tx %s reverted on-chain)", v.Amount, v.Address, txHash)
				continue
			}
			if err := self.backend.WritePayment(v.Address, txHash, v.Amount); err != nil {
				log.Printf("Failed to record payment for %s: %v", v.Address, err)
				return
			}
			log.Printf("Recorded %v Shannon to %s as paid (tx %s already broadcast); not crediting back", v.Amount, v.Address, txHash)
		}
	} else {
		log.Println("No pending payments to resolve")
	}

	// Always clear the lock. A payout can leave it set with no pending payments
	// (e.g. it crashed between locking and recording the debit); without clearing
	// it here that stuck lock would block every future payout with no way out.
	if err := self.backend.UnlockPayouts(); err != nil {
		log.Println("Failed to unlock payouts:", err)
		return
	}

	if self.config.BgSave {
		self.bgSave()
	}
	log.Println("Payouts unlocked")
}

func (self PayoutsProcessor) mustResolvePayout() bool {
	v, _ := strconv.ParseBool(os.Getenv("RESOLVE_PAYOUT"))
	return v
}
