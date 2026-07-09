<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { createPoller } from '../lib/poller';
  import { getPayments } from '../lib/api';
  import { config } from '../lib/config';
  import { formatBalance, formatDate, formatNumber, formatTx } from '../lib/format';

  const poller = createPoller(getPayments);
  onMount(() => poller.start());
  onDestroy(() => poller.stop());

  const d = $derived($poller.data);
</script>

<section class="hero">
  <p>The pool pays transaction fees from its own pocket.</p>
  <p>Total payments sent: <span class="label label-info">{formatNumber(d?.paymentsTotal, '0')}</span></p>
</section>

{#if d?.payments && d.payments.length}
  <h4>Latest Payouts</h4>
  <div class="table-wrap">
    <table class="data">
      <thead><tr><th>Time</th><th>Amount</th><th>Address</th><th>Tx ID</th></tr></thead>
      <tbody>
        {#each d.payments as tx (tx.tx + ':' + tx.address)}
          <tr>
            <td>{formatDate(tx.timestamp)}</td>
            <td>{formatBalance(tx.amount)}</td>
            <td><a class="hash" href={$config.explorerUrl + '/address/' + tx.address} target="_blank" rel="nofollow noreferrer">{tx.address}</a></td>
            <td><a class="hash" href={$config.explorerUrl + '/tx/' + tx.tx} target="_blank" rel="nofollow noreferrer">{formatTx(tx.tx)}</a></td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{:else}
  <h3 class="empty">No payouts yet</h3>
{/if}
