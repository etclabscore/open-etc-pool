<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { createPoller } from '../lib/poller';
  import { getBlocks } from '../lib/api';
  import { config } from '../lib/config';
  import { formatDate, formatNumber, formatPercent } from '../lib/format';
  import BlockRow from './BlockRow.svelte';

  let { tab = 'matured' }: { tab?: string } = $props();

  const poller = createPoller(getBlocks);
  onMount(() => poller.start());
  onDestroy(() => poller.stop());

  const d = $derived($poller.data);
  const luckEntries = $derived(Object.entries(d?.luck ?? {}));

  function variance(shares: number, difficulty: number): number {
    return difficulty ? shares / difficulty : 0;
  }
</script>

<section class="hero">
  <p>Pool always pays full block rewards, including transaction fees and uncle rewards.</p>
  <p>Block maturity requires <span class="label label-success">up to 520</span> blocks. Usually it's fewer.</p>
</section>

{#if luckEntries.length}
  <div class="table-wrap">
    <table class="data">
      <thead><tr><th>Blocks</th><th>Shares/Diff</th><th>Uncle Rate</th><th>Orphan Rate</th></tr></thead>
      <tbody>
        {#each luckEntries as [window, row] (window)}
          <tr>
            <td>{window}</td>
            <td>{formatPercent(row.luck)}</td>
            <td>{formatPercent(row.uncleRate)}</td>
            <td>{formatPercent(row.orphanRate)}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<div class="tabs">
  <a href="#/blocks" class:active={tab === 'matured'}>
    Blocks <span class="badge success">{formatNumber(d?.maturedTotal, '0')}</span>
  </a>
  <a href="#/blocks/immature" class:active={tab === 'immature'}>
    Immature <span class="badge success">{formatNumber(d?.immatureTotal, '0')}</span>
  </a>
  <a href="#/blocks/pending" class:active={tab === 'pending'}>
    New Blocks <span class="badge info">{formatNumber(d?.candidatesTotal, '0')}</span>
  </a>
</div>

{#if tab === 'immature'}
  {#if d?.immature && d.immature.length}
    <h4>Immature Blocks</h4>
    <div class="table-wrap">
      <table class="data">
        <thead><tr><th>Height</th><th>Block Hash</th><th>Time Found</th><th>Variance</th><th>Reward</th><th>Type</th></tr></thead>
        <tbody>
          {#each d.immature as block (block.height + ':' + block.hash)}
            <BlockRow {block} />
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <h3 class="empty">No immature blocks yet</h3>
  {/if}
{:else if tab === 'pending'}
  {#if d?.candidates && d.candidates.length}
    <h4>Recently Found Blocks</h4>
    <div class="table-wrap">
      <table class="data">
        <thead><tr><th>Height</th><th>Time Found</th><th>Variance</th></tr></thead>
        <tbody>
          {#each d.candidates as block (block.height + ':' + block.timestamp)}
            {@const v = variance(block.shares, block.difficulty)}
            <tr>
              <td><a class="hash" href={$config.explorerUrl + '/block/' + block.height} target="_blank" rel="nofollow noreferrer">{formatNumber(block.height)}</a></td>
              <td>{formatDate(block.timestamp)}</td>
              <td><span class="label {v <= 1 ? 'label-success' : 'label-info'}">{formatPercent(v)}</span></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <h3 class="empty">No new blocks yet</h3>
  {/if}
{:else if d?.matured && d.matured.length}
  <h4>Matured Blocks</h4>
  <div class="table-wrap">
    <table class="data">
      <thead><tr><th>Height</th><th>Block Hash</th><th>Time Found</th><th>Variance</th><th>Reward</th><th>Type</th></tr></thead>
      <tbody>
        {#each d.matured as block (block.height + ':' + block.hash)}
          <BlockRow {block} />
        {/each}
      </tbody>
    </table>
  </div>
{:else}
  <h3 class="empty">No matured blocks yet</h3>
{/if}
