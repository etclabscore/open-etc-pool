<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { createPoller } from '../lib/poller';
  import { getMiners } from '../lib/api';
  import { formatHashrate, formatDate, formatNumber } from '../lib/format';

  const poller = createPoller(getMiners);
  onMount(() => poller.start());
  onDestroy(() => poller.stop());

  const d = $derived($poller.data);
  // Miners map -> array, sorted by hashrate descending (matches the Ember route).
  const miners = $derived(
    Object.entries(d?.miners ?? {})
      .map(([login, miner]) => ({ login, ...miner }))
      .sort((a, b) => b.hr - a.hr),
  );
</script>

<section class="hero">
  <p>Total hashrate: <strong>{formatHashrate(d?.hashrate)}</strong>.</p>
  <p>Total miners: <span class="label label-info">{formatNumber(d?.minersTotal, '0')}</span></p>
</section>

{#if miners.length}
  <h4>Miners</h4>
  <div class="table-wrap">
    <table class="data">
      <thead><tr><th>Login</th><th>Hashrate</th><th>Last Beat</th></tr></thead>
      <tbody>
        {#each miners as m (m.login)}
          <tr class={m.offline ? 'warning' : ''}>
            <td><a class="hash" href={'#/account/' + m.login}>{m.login}</a></td>
            <td>{formatHashrate(m.hr)}</td>
            <td>{formatDate(m.lastBeat)}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{:else}
  <h3 class="empty">No miners</h3>
{/if}
