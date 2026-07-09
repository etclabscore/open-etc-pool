import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark';

function initial(): Theme {
  if (typeof document === 'undefined') return 'light';
  const forced = document.documentElement.getAttribute('data-theme');
  if (forced === 'light' || forced === 'dark') return forced;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export const theme = writable<Theme>(initial());

export function toggleTheme(): void {
  theme.update((current) => {
    const next: Theme = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try {
      localStorage.setItem('theme', next);
    } catch {
      // ignore storage failures (private mode, etc.)
    }
    return next;
  });
}
