import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// The pool API is same-origin in production (nginx proxies /api to the pool
// binary). In dev we proxy /api to a locally running pool on :8080.
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
});
