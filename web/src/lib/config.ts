import { writable } from 'svelte/store';

// Runtime configuration, fetched from /config.json at startup so operators can
// change endpoints without rebuilding the SPA. Falls back to these defaults.
export interface AppConfig {
  apiUrl: string;
  stratumHost: string;
  stratumPort: number;
  httpHost: string;
  httpPort: number;
  network: string; // 'classic' | 'mordor'
  poolFee: string;
  payoutThreshold: string;
  blockTime: number; // seconds between blocks, for network hashrate
  explorerUrl: string; // block/tx explorer base, no trailing slash
}

export const defaults: AppConfig = {
  apiUrl: '/',
  stratumHost: 'localhost',
  stratumPort: 8008,
  httpHost: 'http://localhost',
  httpPort: 8888,
  network: 'classic',
  poolFee: '1%',
  payoutThreshold: '0.5 ETC',
  blockTime: 13.2,
  explorerUrl: 'https://expedition.dev',
};

export const config = writable<AppConfig>(defaults);

export async function loadConfig(): Promise<void> {
  try {
    const res = await fetch(import.meta.env.BASE_URL + 'config.json', { cache: 'no-cache' });
    if (res.ok) {
      const loaded = (await res.json()) as Partial<AppConfig>;
      config.set({ ...defaults, ...loaded });
    }
  } catch {
    // keep defaults if config.json is missing or unparseable
  }
}
