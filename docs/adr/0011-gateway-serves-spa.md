# 0011 — Gateway serves the SPA

- Status: Accepted
- Date: 2026-06-19

## Context

The gateway is the single ingress (ADR-0010). The browser needs both the SPA
and the API from one origin — simplest for CORS, cookies, CSP, and the
air-gapped single-entry-point model.

## Decision

The gateway serves the built SPA's static assets from `WEB_DIR` with
**history-API fallback**: unknown paths return `index.html` so the client-side
router handles deep links. API (`/api/`) and health paths are skipped so they
fall through to their handlers; the static handler is mounted last.

- In the container image, a multi-stage build compiles the SPA (node), builds
  the gateway (go), and the final image serves `dist` from `/web`
  (`WEB_DIR=/web`).
- In dev, `WEB_DIR` is empty; the SPA runs on the Vite dev server, which
  proxies `/api` and `/config.json` to the gateway.
- **Runtime config:** the gateway serves `GET /config.json` (OIDC authority +
  client id from its env). The SPA fetches it at startup, so the built bundle is
  environment-agnostic — one image runs across deployments without rebuilding.
  In dev the SPA falls back to build-time `VITE_*` if `/config.json` is absent.

## Consequences

- One origin for UI + API; no separate static host or CDN (air-gap friendly).
- Deep links (e.g. `/tickets/:id`) work on full page load via the fallback.
- The gateway image now depends on the web build; CI builds the SPA separately
  (the `web` job) and the image build repeats it in its node stage.
- Static assets are served by Fiber's `filesystem` middleware (fasthttp-native);
  no net/http conversion on the hot path.
