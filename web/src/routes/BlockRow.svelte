<script lang="ts">
  import { config } from '../lib/config';
  import type { Block } from '../lib/types';
  import { formatDate, formatNumber, formatPercent, formatReward } from '../lib/format';

  let { block }: { block: Block } = $props();

  const variance = $derived(block.difficulty ? block.shares / block.difficulty : 0);
  const isLucky = $derived(variance <= 1.0);
  const isOk = $derived(!block.orphan);
  const reward = $derived(formatReward(block.reward, block.orphan));
  const heightUrl = $derived(
    block.uncle
      ? $config.explorerUrl + '/uncle/' + block.uncleHeight
      : $config.explorerUrl + '/block/' + block.height,
  );
</script>

<tr>
  <td><a class="hash" href={heightUrl} target="_blank" rel="nofollow noreferrer">{formatNumber(block.height)}</a></td>
  <td>
    {#if block.uncle}
      <a class="hash" href={$config.explorerUrl + '/uncle/' + block.hash} target="_blank" rel="nofollow noreferrer">{block.hash}</a>
    {:else if block.orphan}
      <span class="label label-danger">Orphan</span>
    {:else}
      <a class="hash" href={$config.explorerUrl + '/block/' + block.hash} target="_blank" rel="nofollow noreferrer">{block.hash}</a>
    {/if}
  </td>
  <td>{formatDate(block.timestamp)}</td>
  <td><span class="label {isLucky ? 'label-success' : 'label-info'}">{formatPercent(variance)}</span></td>
  <td>
    {#if block.uncle}<span class="label label-default">{reward}</span>
    {:else if isOk}<span class="label label-primary">{reward}</span>{/if}
  </td>
  <td>
    {#if block.uncle}<span class="label label-default">Uncle</span>
    {:else if isOk}<span class="label label-primary">Block</span>{/if}
  </td>
</tr>
