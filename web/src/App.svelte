<script lang="ts">
  import { onMount } from 'svelte';
  import Nav from './lib/Nav.svelte';
  import Footer from './lib/Footer.svelte';
  import { statsPoller } from './lib/stores';
  import Home from './routes/Home.svelte';
  import Account from './routes/Account.svelte';
  import Blocks from './routes/Blocks.svelte';
  import Miners from './routes/Miners.svelte';
  import Payments from './routes/Payments.svelte';
  import Help from './routes/Help.svelte';
  import About from './routes/About.svelte';
  import NotFound from './routes/NotFound.svelte';

  function currentPath(): string {
    return location.hash.replace(/^#/, '') || '/';
  }

  let path = $state(currentPath());

  onMount(() => {
    if (!location.hash) location.hash = '#/';
    const onHash = () => {
      path = currentPath();
      window.scrollTo(0, 0);
    };
    window.addEventListener('hashchange', onHash);
    statsPoller.start();
    return () => window.removeEventListener('hashchange', onHash);
  });

  // Minimal hash router. `key` drives remounting: it stays stable across tab
  // switches within blocks/account so those don't refetch, but changes when the
  // account address changes.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  interface Routed {
    component: any;
    props: Record<string, unknown>;
    key: string;
  }

  const routed = $derived.by((): Routed => {
    const seg = path.split('/').filter(Boolean);
    if (seg.length === 0) return { component: Home, props: {}, key: 'home' };
    switch (seg[0]) {
      case 'account':
        if (seg[1]) {
          const login = seg[1].toLowerCase();
          return {
            component: Account,
            props: { login, tab: seg[2] === 'payouts' ? 'payouts' : 'workers' },
            key: 'account/' + login,
          };
        }
        return { component: NotFound, props: {}, key: 'notfound' };
      case 'blocks': {
        const tab = seg[1] === 'immature' ? 'immature' : seg[1] === 'pending' ? 'pending' : 'matured';
        return { component: Blocks, props: { tab }, key: 'blocks' };
      }
      case 'miners':
        return { component: Miners, props: {}, key: 'miners' };
      case 'payments':
        return { component: Payments, props: {}, key: 'payments' };
      case 'help':
        return { component: Help, props: {}, key: 'help' };
      case 'about':
        return { component: About, props: {}, key: 'about' };
      default:
        return { component: NotFound, props: {}, key: 'notfound' };
    }
  });
</script>

<Nav {path} />
<main class="container page">
  {#key routed.key}
    {@const Cmp = routed.component}
    <Cmp {...routed.props} />
  {/key}
</main>
<Footer />
