**First of all make sure your Redis instance and backups are configured properly http://redis.io/topics/persistence.**

Keep in mind that pool maintains all balances in **Shannon**.

# Processing and Resolving Payouts

**You MUST run payouts module in a separate process**, ideally don't run it as daemon and process payouts 2-3 times per day and watch how it goes. **You must configure logging**, otherwise it can lead to big problems.

Module will fetch accounts and sequentially process payouts.

For every account who reached minimal threshold:

* Check if we have enough peers on a node
* Check that account is unlocked

If any of checks fails, module will not even try to continue.

* Check if we have enough money for payout (should not happen under normal circumstances)
* Lock payments

If payments can't be locked (another lock exist, usually after a failure) module will halt payouts.

* Read the pool account nonce (via `eth_getTransactionCount`) and record it
* Deduct the miner's balance and log a pending payment
* Submit the transaction via `eth_sendTransaction`
* Record the returned TX hash, then move the payment from pending to paid and unlock

The nonce is recorded **before** the transaction is sent, so an interrupted payout
can later be reconciled against the chain without double-paying (see below).

**If transaction submission fails, payouts remain locked and halted** until you
resolve them. On the next normal start the module also refuses to run while any
pending payment exists.

And so on. Repeat for every account.

After payout session, payment module will perform `BGSAVE` (background saving) on Redis if you have enabled `bgsave` option.

## Resolving Locked Payouts (automatic)

If a payout was interrupted, restart the payouts module in maintenance mode with
the `RESOLVE_PAYOUT=1` (or `RESOLVE_PAYOUT=True`) environment variable:

`RESOLVE_PAYOUT=1 ./open-etc-pool payouts.json`.

It fetches the pending payment(s) from `<coin>:payments:pending` — normally a
single entry — and **reconciles each against the chain instead of blindly
crediting it back**, so it never double-pays a payout that was actually broadcast:

* Using the recorded **nonce**, it checks `eth_getTransactionCount`:
  * nonce **mined** → the payout went out → recorded as paid (credited back only
    if the transaction reverted on-chain);
  * nonce **still in the mempool** → left in place; re-run `RESOLVE_PAYOUT=1` once
    it mines;
  * nonce **never used** → nothing was sent → balance credited back.
* For payments recorded before this version (no nonce), it falls back to the
  recorded TX hash: present (and not reverted) → treated as paid; absent →
  credited back.

`No pending payments to resolve` means there is nothing to fix. When it finishes
it unlocks payouts and halts:

```
Payouts unlocked
Now you have to restart payouts module with RESOLVE_PAYOUT=0 for normal run
```

Then run payouts normally again (unset `RESOLVE_PAYOUT` or set `RESOLVE_PAYOUT=0`).

## Resolving Failed Payment (manual)

You can perform manual maintenance using `geth` and `redis-cli` utilities.

### Check For Failed Transactions:

Perform the following command in a `redis-cli`:

```
ZREVRANGE "eth:payments:pending" 0 -1 WITHSCORES
```

Result will be like this:

> 1) "0xb85150eb365e7df0941f0cf08235f987ba91506a:25000000"

It's a pair of `LOGIN:AMOUNT`.

>2) "1462920526"

It's a `UNIXTIME`

### Manual Payment Submission

**Make sure there is no TX sent using block explorer. Skip this step if payment actually exist in a blockchain.**

```javascript
eth.sendTransaction({
  from: eth.coinbase,
  to: '0xb85150eb365e7df0941f0cf08235f987ba91506a',
  value: web3.toWei(25000000, 'shannon')
})

// => 0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331
```

**Write down tx hash**.

### Store Payment in Redis

Also usable for fixing missing payment entries.

```
ZADD "eth:payments:all" 1462920526 0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331:0xb85150eb365e7df0941f0cf08235f987ba91506a:25000000
```

```
ZADD "eth:payments:0xb85150eb365e7df0941f0cf08235f987ba91506a" 1462920526 0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331:25000000
```

### Delete Erroneous Payment Entry

```
ZREM "eth:payments:pending" "0xb85150eb365e7df0941f0cf08235f987ba91506a:25000000"
```

### Update Internal Stats

```
HINCRBY "eth:finances" pending -25000000
HINCRBY "eth:finances" paid 25000000
```

### Unlock Payouts

```
DEL "eth:payments:lock"
```

## Resolving Missing Payment Entries

If pool actually paid but didn't log transaction, scroll up to `Store Payment in Redis` section. You should have a transaction hash from block explorer.

## Transaction Didn't Confirm

If you are sure, just repeat it manually, you should have all the logs.
