<script lang="ts">
  import Icon from './Icon.svelte';
  import Logo from './Logo.svelte';
  import { statsPoller } from './stores';
  import { theme, toggleTheme } from './theme';

  let { path }: { path: string } = $props();

  const items = [
    { href: '#/', label: 'Home', icon: 'home', match: (p: string) => p === '/' },
    { href: '#/blocks', label: 'Pool Blocks', icon: 'blocks', match: (p: string) => p.startsWith('/blocks') },
    { href: '#/payments', label: 'Payments', icon: 'send', match: (p: string) => p.startsWith('/payments') },
    { href: '#/miners', label: 'Miners', icon: 'users', match: (p: string) => p.startsWith('/miners') },
    { href: '#/help', label: 'Help', icon: 'rocket', match: (p: string) => p.startsWith('/help') },
    { href: '#/about', label: 'About', icon: 'about', match: (p: string) => p.startsWith('/about') },
  ];

  // Immature badge = immature + candidate blocks, from the shared pool stats.
  const immature = $derived(
    ($statsPoller.data?.immatureTotal ?? 0) + ($statsPoller.data?.candidatesTotal ?? 0),
  );
</script>

<nav class="navbar">
  <div class="container navbar-inner">
    <a class="navbar-brand" href="#/">
      <Logo size={28} />
      <span>ETC&nbsp;Pool</span>
    </a>
    <div class="nav-links">
      {#each items as item (item.href)}
        <a href={item.href} class:active={item.match(path)}>
          <Icon name={item.icon} />
          <span class="nav-label">{item.label}</span>
          {#if item.label === 'Pool Blocks' && immature > 0}
            <span class="badge success">{immature}</span>
          {/if}
        </a>
      {/each}
      <button
        class="theme-toggle"
        type="button"
        onclick={toggleTheme}
        aria-label="Toggle dark mode"
        title="Toggle dark mode">
        <Icon name={$theme === 'dark' ? 'sun' : 'moon'} />
      </button>
    </div>
  </div>
</nav>
