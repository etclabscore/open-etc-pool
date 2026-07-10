# open-etc-pool frontend

The pool's web UI: a [Vite](https://vitejs.dev) + [Svelte 5](https://svelte.dev) +
TypeScript single-page app. It polls the pool's read-only JSON API every 5s and
renders stats, blocks, payments, miners, and per-address account pages, with
light and dark themes.

## Develop

```sh
npm ci
npm run dev      # dev server on :5173, proxies /api -> http://localhost:8080
```

`npm run check` type-checks (svelte-check), `npm test` runs the unit tests
(vitest), `npm run build` produces the static site in `dist/`.

## Runtime configuration

The app fetches [`public/config.json`](public/config.json) at startup, so an
operator can change endpoints and labels **without rebuilding**:

| key | meaning |
|---|---|
| `apiUrl` | base URL of the pool API (`/` when nginx proxies same-origin) |
| `explorerUrl` | block/tx explorer base (e.g. `https://expedition.dev`) |
| `stratumHost` / `stratumPort` | shown on the Help / Home pages |
| `network` | `classic` or `mordor` (drives the testnet warning) |
| `poolFee`, `payoutThreshold`, `blockTime` | display / network-hashrate values |

`apiUrl` and `explorerUrl` are validated to be `http(s)`/relative before use.

## Deploy

`npm run build`, then serve `dist/` as static files with nginx and proxy `/api`
to the pool — see [`../misc/nginx-default.conf`](../misc/nginx-default.conf). The
app uses hash-based routing, so no server rewrite rules are needed. If you serve
the API from a different origin, add that origin to the CSP `connect-src`.
