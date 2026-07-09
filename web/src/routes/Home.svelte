<script lang="ts">
  import Icon from '../lib/Icon.svelte';
  import { config } from '../lib/config';
  import { statsPoller } from '../lib/stores';
  import { formatHashrate, formatNumber, formatPercent, formatRelative, metricPrefix, secondsToMs } from '../lib/format';

  const stats = $derived($statsPoller.data);

  const bestNode = $derived.by(() => {
    const nodes = stats?.nodes ?? [];
    let best = nodes[0];
    for (const n of nodes) {
      if (Number(n.height) > Number(best?.height ?? 0)) best = n;
    }
    return best;
  });

  const height = $derived(Number(bestNode?.height ?? 0));
  const difficulty = $derived(Number(bestNode?.difficulty ?? 0));
  const networkHashrate = $derived(difficulty / ($config.blockTime || 1));
  const roundShares = $derived(Number(stats?.stats?.roundShares ?? 0));
  const roundVariance = $derived(difficulty ? roundShares / difficulty : 0);
  const lastBlockFound = $derived(Number(stats?.stats?.lastBlockFound ?? 0));

  let login = $state(typeof localStorage !== 'undefined' ? (localStorage.getItem('login') ?? '') : '');

  function lookup(e: Event) {
    e.preventDefault();
    const l = login.trim().toLowerCase();
    if (!l) return;
    localStorage.setItem('login', l);
    location.hash = '#/account/' + l;
  }
</script>

{#if !stats && $statsPoller.error}
  <div class="callout callout-danger">
    <h4>Stats API Temporarily Down</h4>
    <p>Usually it's just a temporary issue and mining is not affected.</p>
  </div>
{/if}

<section class="hero">
  <div class="grid-3">
    <div>
      <h1 class="brand-title">ETC Pool</h1>
      <p>
        Min. payout threshold: <strong>{$config.payoutThreshold}</strong>. Payouts are continuous
        throughout the day.
      </p>
      <p>
        <span class="label label-success">PROP</span>
        Stable and profitable pool with regular payouts.
      </p>
    </div>

    <div class="stats">
      <div class="stat"><Icon name="users" /><span class="stat-label">Miners Online</span><span class="stat-value">{formatNumber(stats?.minersTotal, '0')}</span></div>
      <div class="stat"><Icon name="gauge" /><span class="stat-label">Pool Hash Rate</span><span class="stat-value">{formatHashrate(stats?.hashrate)}</span></div>
      <div class="stat"><Icon name="money" /><span class="stat-label">Pool Fee</span><span class="stat-value"><span class="label label-success">{$config.poolFee}</span></span></div>
      {#if lastBlockFound}
        <div class="stat"><Icon name="clock" /><span class="stat-label">Last Block Found</span><span class="stat-value">{formatRelative(secondsToMs(lastBlockFound))}</span></div>
      {/if}
    </div>

    <div class="stats">
      <div class="stat"><Icon name="unlock" /><span class="stat-label">Network Difficulty</span><span class="stat-value">{metricPrefix(difficulty)}</span></div>
      <div class="stat"><Icon name="gauge" /><span class="stat-label">Network Hash Rate</span><span class="stat-value">{formatHashrate(networkHashrate)}</span></div>
      <div class="stat"><Icon name="height" /><span class="stat-label">Blockchain Height</span><span class="stat-value">{formatNumber(height, '0')}</span></div>
      <div class="stat"><Icon name="clock" /><span class="stat-label">Current Round Variance</span><span class="stat-value">{formatPercent(roundVariance)}</span></div>
    </div>
  </div>
</section>

<h4>Your Stats &amp; Payment History</h4>
<form class="input-group" onsubmit={lookup}>
  <input
    type="text"
    bind:value={login}
    placeholder="Enter Your Ethereum Classic Address"
    spellcheck="false"
    autocapitalize="off" />
  <button class="btn btn-etc" type="submit"><Icon name="search" /> Lookup</button>
</form>

{#if $config.network === 'mordor'}
  <div class="callout callout-danger">
    <h4>Warning</h4>
    <p>
      This pool runs on the experimental <strong>Mordor</strong> testnet. Coins mined here have no
      value and exist only for testing.
    </p>
  </div>
{/if}

<section class="hero instructions">
  <h3>Instructions</h3>
  <p>
    Point your miner at
    <a href="#/help"><span class="ul ul-danger">stratum</span></a>
    <code>{$config.stratumHost}:{$config.stratumPort}</code>
    — see the <a href="#/help">Help</a> page for full setup.
  </p>
</section>
