import { writable, type Readable } from 'svelte/store';

export interface PollState<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
}

export interface Poller<T> extends Readable<PollState<T>> {
  start(): void;
  stop(): void;
  refresh(): Promise<void>;
}

// A polling store: refetches every intervalMs (5s to match the pool's stats
// cadence). The last good data is kept across transient errors so the UI does
// not flicker to an empty state on a single failed poll.
export function createPoller<T>(fetcher: () => Promise<T>, intervalMs = 5000): Poller<T> {
  const store = writable<PollState<T>>({ data: null, error: null, loading: true });
  let timer: ReturnType<typeof setInterval> | undefined;
  let stopped = true;

  async function tick(): Promise<void> {
    try {
      const data = await fetcher();
      if (!stopped) store.set({ data, error: null, loading: false });
    } catch (e) {
      if (!stopped) store.update((s) => ({ ...s, error: String(e), loading: false }));
    }
  }

  return {
    subscribe: store.subscribe,
    refresh: tick,
    start() {
      if (!stopped) return;
      stopped = false;
      void tick();
      timer = setInterval(tick, intervalMs);
    },
    stop() {
      stopped = true;
      if (timer) clearInterval(timer);
    },
  };
}
