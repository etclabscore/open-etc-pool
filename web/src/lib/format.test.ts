import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  formatHashrate,
  metricPrefix,
  formatBalance,
  formatReward,
  formatTx,
  formatPercent,
  formatNumber,
  formatRelative,
} from './format';

describe('formatHashrate', () => {
  it('scales to the right unit', () => {
    expect(formatHashrate(0)).toBe('0.00 H');
    expect(formatHashrate(999)).toBe('999.00 H');
    expect(formatHashrate(1500)).toBe('1.50 KH');
    expect(formatHashrate(3_520_000_000_000)).toBe('3.52 TH');
  });
  it('handles missing/garbage input', () => {
    expect(formatHashrate(undefined)).toBe('0.00 H');
    expect(formatHashrate('nope')).toBe('0.00 H');
  });
});

describe('metricPrefix', () => {
  it('leaves small numbers untouched', () => {
    expect(metricPrefix(500)).toBe('500');
  });
  it('adds a metric prefix', () => {
    expect(metricPrefix(325_000_000_000_000)).toBe('325.000 T');
  });
});

describe('formatBalance (Shannon -> ETC)', () => {
  it('converts with 8 decimals', () => {
    expect(formatBalance(234500000)).toBe('0.23450000');
    expect(formatBalance(0)).toBe('0.00000000');
  });
});

describe('formatReward (Wei -> ETC)', () => {
  it('converts with 6 decimals', () => {
    expect(formatReward('3200000000000000000')).toBe('3.200000');
  });
  it('is 0 for orphans', () => {
    expect(formatReward('3200000000000000000', true)).toBe('0');
  });
});

describe('formatTx', () => {
  it('truncates head...tail', () => {
    const tx = '0x' + 'a'.repeat(64);
    expect(formatTx(tx)).toBe('a'.repeat(24) + '...' + 'a'.repeat(24));
  });
  it('handles empty input', () => {
    expect(formatTx('')).toBe('');
    expect(formatTx(undefined)).toBe('');
  });
});

describe('formatPercent', () => {
  it('formats a fraction as a percent', () => {
    expect(formatPercent(0.83)).toBe('83%');
    expect(formatPercent(1.26)).toBe('126%');
  });
});

describe('formatNumber', () => {
  it('groups thousands', () => {
    expect(formatNumber(19502345)).toBe('19,502,345');
  });
  it('uses the fallback for empty values', () => {
    expect(formatNumber(undefined, '0')).toBe('0');
    expect(formatNumber('', '0')).toBe('0');
  });
});

describe('formatRelative', () => {
  afterEach(() => vi.useRealTimers());
  it('renders relative to now', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-01-01T00:00:00Z'));
    expect(formatRelative(Date.now() - 5 * 60_000)).toBe('5 minutes ago');
    expect(formatRelative(Date.now() + 2 * 3_600_000)).toBe('in 2 hours');
  });
  it('handles missing input', () => {
    expect(formatRelative(undefined)).toBe('—');
  });
});
