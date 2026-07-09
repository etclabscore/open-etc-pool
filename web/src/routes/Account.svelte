<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Icon from '../lib/Icon.svelte';
  import { config } from '../lib/config';
  import { statsPoller } from '../lib/stores';
  import { getAccount, NotFoundError } from '../lib/api';
  import type { AccountResponse } from '../lib/types';
  import {
    formatBalance,
    formatHashrate,
    formatNumber,
    formatPercent,
    formatRelative,
    formatDate,
    secondsToMs,
  } from '../lib/format';

  let { login, tab = 'workers' }: { login: string; tab?: string } = $props();

  let data = $state<AccountResponse | null>(null);
  let notFound = $state(false);
  let timer: ReturnType<typeof setInterval> | undefined;

  async function load() {
    try {
      data = await getAccount(login);
      notFound = false;
    } catch (e) {
      if (e instanceof NotFoundError) {
        notFound = true;
        data = null;
      }
      // transient errors: keep the last good data
    }
  }

  onMount(() => {
    void load();
    timer = setInterval(load, 5000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  const s = $derived(data?.stats ?? {});

  const poolRoundShares = $derived(Number($statsPoller.data?.stats?.roundShares ?? 0));
  const roundPercent = $derived(poolRoundShares ? Number(data?.roundShares ?? 0) / poolRoundShares : 0);

  // Next ECIP-1099 epoch switch (every 60000 blocks), as a future timestamp.
  const nextEpoch = $derived.by(() => {
    const nodes = $statsPoller.data?.nodes ?? [];
    let h = 0;
    for (const n of nodes) h = Math.max(h, Number(n.height));
    const remaining = 60000 - (h % 60000);
    return Date.now() + remaining * 1000 * ($config.blockTime || 1);
  });

  const workerEntries = $derived(Object.entries(data?.workers ?? {}));
  const apiBase = $derived($config.apiUrl.endsWith('/') ? $config.apiUrl : $config.apiUrl + '/');
</script>

{#if notFound}
  <div class="callout callout-danger">
    <h1>No Account Data Available</h1>
    <p>If you are looking for your account stats, you need to submit at least a single share.</p>
    <p><a href="#/">Back to the pool home</a></p>
  </div>
{:else if data}
  <section class="hero">
    <div class="account-address hash">{login}</div>
    <div class="grid-3">
      <div class="stats">
        <div class="stat"><Icon name="cloud" /><span class="stat-label">Immature Balance</span><span class="stat-value">{formatBalance(s.immature)}</span></div>
        <div class="stat"><Icon name="bank" /><span class="stat-label">Pending Balance</span><span class="stat-value">{formatBalance(s.balance)}</span></div>
        {#if s.pending}
          <div class="stat"><Icon name="clock" /><span class="stat-label">Current Payment</span><span class="stat-value">{formatBalance(s.pending)}</span></div>
        {/if}
        <div class="stat"><Icon name="money" /><span class="stat-label">Total Paid</span><span class="stat-value">{formatBalance(s.paid)}</span></div>
      </div>

      <div class="stats">
        {#if s.lastShare}
          <div class="stat"><Icon name="clock" /><span class="stat-label">Last Share Submitted</span><span class="stat-value">{formatRelative(secondsToMs(s.lastShare))}</span></div>
        {/if}
        <div class="stat"><Icon name="gears" /><span class="stat-label">Workers Online</span><span class="stat-value">{formatNumber(data.workersOnline, '0')}</span></div>
        <div class="stat"><Icon name="gauge" /><span class="stat-label">Hashrate (30m)</span><span class="stat-value">{formatHashrate(data.currentHashrate)}</span></div>
        <div class="stat"><Icon name="gauge" /><span class="stat-label">Hashrate (3h)</span><span class="stat-value">{formatHashrate(data.hashrate)}</span></div>
      </div>

      <div class="stats">
        <div class="stat"><Icon name="height" /><span class="stat-label">Blocks Found</span><span class="stat-value">{formatNumber(s.blocksFound, '0')}</span></div>
        <div class="stat"><Icon name="send" /><span class="stat-label">Total Payments</span><span class="stat-value">{formatNumber(data.paymentsTotal, '0')}</span></div>
        <div class="stat"><Icon name="gears" /><span class="stat-label">Your Round Share</span><span class="stat-value">{formatPercent(roundPercent, 6)}</span></div>
        <div class="stat"><Icon name="clock" /><span class="stat-label">Epoch Switch</span><span class="stat-value">{formatRelative(nextEpoch)}</span></div>
      </div>
    </div>
  </section>

  <div class="tabs">
    <a href={'#/account/' + login} class:active={tab === 'workers'}>
      Workers
      {#if data.workersOffline > 0}<span class="badge danger">{data.workersOffline}</span>{/if}
    </a>
    <a href={'#/account/' + login + '/payouts'} class:active={tab === 'payouts'}>Payouts</a>
  </div>

  {#if tab === 'payouts'}
    {#if data.payments && data.payments.length}
      <h4>Your Latest Payouts</h4>
      <div class="table-wrap">
        <table class="data">
          <thead><tr><th>Time</th><th>Tx ID</th><th>Amount</th></tr></thead>
          <tbody>
            {#each data.payments as tx (tx.tx)}
              <tr>
                <td>{formatDate(tx.timestamp)}</td>
                <td><a class="hash" href={$config.explorerUrl + '/tx/' + tx.tx} target="_blank" rel="nofollow noreferrer">{tx.tx}</a></td>
                <td>{formatBalance(tx.amount)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <h3 class="empty">No payouts yet</h3>
    {/if}
  {:else if workerEntries.length}
    <h4>Your Workers</h4>
    <div class="table-wrap">
      <table class="data">
        <thead>
          <tr>
            <th>ID</th>
            <th>Hashrate (rough, short average)</th>
            <th>Hashrate (accurate, long average)</th>
            <th>Last Share</th>
          </tr>
        </thead>
        <tbody>
          {#each workerEntries as [id, w] (id)}
            <tr class={w.offline ? 'warning' : 'success'}>
              <td>{id}</td>
              <td>{formatHashrate(w.hr)}</td>
              <td>{formatHashrate(w.hr2)}</td>
              <td>{formatRelative(secondsToMs(w.lastBeat))}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <div class="alert alert-info">
      Hashrate is smoothed over two windows: a short one (~30 min) and a long one (~3 h). A worker is
      flagged offline when it hasn't submitted a share for half the short window.
    </div>
    <div class="alert alert-info">
      Your bulk stats JSON API URL:
      <a class="hash" href={apiBase + 'api/accounts/' + login} target="_blank" rel="noreferrer">{apiBase}api/accounts/{login}</a>
    </div>
  {:else}
    <h3 class="empty">No workers online</h3>
  {/if}
{:else}
  <p class="loading">Loading account…</p>
{/if}
