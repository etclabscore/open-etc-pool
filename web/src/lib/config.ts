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

// sanitizeUrl only accepts http(s) or same-origin-relative URLs. config.json is a
// separate file (often on a looser-ACL host than the JS bundle), and its apiUrl /
// explorerUrl flow into fetch targets and <a href>. Without this, a tampered
// config could inject a `javascript:` URL that runs script in the pool's origin
// when a block/tx link is clicked.
export function sanitizeUrl(value: unknown, fallback: string): string {
  if (typeof value !== 'string' || value === '') return fallback;
  // A base is only needed to resolve relative values like "/"; the scheme check
  // is what matters. Falls back to a dummy base when there is no window (tests).
  const base = typeof window !== 'undefined' ? window.location.origin : 'http://localhost';
  try {
    const u = new URL(value, base);
    if (u.protocol === 'http:' || u.protocol === 'https:') return value;
  } catch {
    // not a parseable URL
  }
  return fallback;
}

export async function loadConfig(): Promise<void> {
  try {
    const res = await fetch(import.meta.env.BASE_URL + 'config.json', { cache: 'no-cache' });
    if (res.ok) {
      const loaded = (await res.json()) as Partial<AppConfig>;
      config.set({
        ...defaults,
        ...loaded,
        apiUrl: sanitizeUrl(loaded.apiUrl, defaults.apiUrl),
        explorerUrl: sanitizeUrl(loaded.explorerUrl, defaults.explorerUrl),
      });
    }
  } catch {
    // keep defaults if config.json is missing or unparseable
  }
}
