// Formatting helpers ported 1:1 from the Ember frontend's helpers so numbers,
// hashrates, balances and timestamps render identically.

const HR_UNITS = ['H', 'KH', 'MH', 'GH', 'TH', 'PH'];

// H/s -> human hashrate, e.g. 12.34 MH. (Ember: helpers/format-hashrate.js)
export function formatHashrate(v: number | string | undefined | null): string {
  let n = Number(v) || 0;
  let i = 0;
  while (n > 1000 && i < HR_UNITS.length - 1) {
    n /= 1000;
    i++;
  }
  return n.toFixed(2) + ' ' + HR_UNITS[i];
}

const METRIC_UNITS = ['K', 'M', 'G', 'T', 'P'];

// Large plain number -> metric prefix, e.g. difficulty. (Ember: with-metric-prefix)
export function metricPrefix(v: number | string | undefined | null): string {
  let n = Number(v) || 0;
  if (n < 1000) return String(n);
  let i = 0;
  while (n > 1000 && i < METRIC_UNITS.length) {
    n /= 1000;
    i++;
  }
  return n.toFixed(3) + ' ' + METRIC_UNITS[i - 1];
}

// Shannon (Gwei) -> ETC with 8 decimals. (Ember: format-balance, payment.formatAmount)
export function formatBalance(v: number | string | undefined | null): string {
  return (Number(v) * 1e-9).toFixed(8);
}

// Wei -> ETC with 6 decimals, for block rewards. (Ember: block.formatReward)
export function formatReward(reward: string | number | undefined | null, orphan = false): string {
  if (orphan) return '0';
  return (Number(reward) * 1e-18).toFixed(6);
}

// Unix seconds -> locale date/time string. (Ember: format-date-locale)
export function formatDate(tsSeconds: number | string | undefined | null): string {
  const n = Number(tsSeconds);
  if (!n) return '—';
  return new Date(n * 1000).toLocaleString();
}

// Absolute ms timestamp -> relative time, e.g. "5 minutes ago" / "in 2 hours".
// (Ember used ember-intl format-relative with ms input.)
const rtf = new Intl.RelativeTimeFormat('en-US', { numeric: 'auto' });
export function formatRelative(ms: number | string | undefined | null): string {
  const target = Number(ms);
  if (!target) return '—';
  const diff = target - Date.now();
  const abs = Math.abs(diff);
  if (abs < 60_000) return rtf.format(Math.round(diff / 1000), 'second');
  if (abs < 3_600_000) return rtf.format(Math.round(diff / 60_000), 'minute');
  if (abs < 86_400_000) return rtf.format(Math.round(diff / 3_600_000), 'hour');
  return rtf.format(Math.round(diff / 86_400_000), 'day');
}

// Seconds -> ms, for feeding formatRelative. (Ember: seconds-to-ms)
export function secondsToMs(v: number | string | undefined | null): number {
  return Number(v) * 1000;
}

// Truncate a 0x tx hash to head…tail. (Ember: format-tx)
export function formatTx(tx: string | undefined | null): string {
  if (!tx) return '';
  return tx.substring(2, 26) + '...' + tx.substring(42);
}

const numberFmt = new Intl.NumberFormat('en-US');

// Grouped integer/number. (Ember: ember-intl format-number)
export function formatNumber(v: number | string | undefined | null, fallback = ''): string {
  if (v === undefined || v === null || v === '') return fallback;
  const n = Number(v);
  if (Number.isNaN(n)) return fallback;
  return numberFmt.format(n);
}

// Fraction -> percent, e.g. 0.83 -> "83%". (Ember: format-number style='percent')
export function formatPercent(v: number | string | undefined | null, maxFractionDigits = 2): string {
  const n = Number(v) || 0;
  return new Intl.NumberFormat('en-US', {
    style: 'percent',
    maximumFractionDigits: maxFractionDigits,
  }).format(n);
}
