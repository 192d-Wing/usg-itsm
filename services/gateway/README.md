# gateway

The TLS edge / BFF for USG-ITSM. Terminates **TLS 1.3**, validates OIDC bearer
tokens, enforces coarse auth, routes to backend services, and (later) serves the
SPA.

## Responsibilities

- TLS 1.3 termination (dev: in-memory self-signed; prod: mounted cert).
- OIDC token validation on `/api/v1/*` (independent of each service —
  defense in depth, ADR-0005).
- **Reverse-proxy routing** to backend services over TLS 1.3 (ADR-0010),
  forwarding the `Authorization` header so upstreams re-validate.

## Routes

| Path                      | Target                          |
|---------------------------|---------------------------------|
| `GET /healthz`, `/readyz` | gateway (public)                |
| `GET /config.json`        | gateway (public SPA runtime config: OIDC authority + client id) |
| `GET /api/v1/me`          | gateway (returns caller claims) |
| `/api/v1/tickets[/*]`     | ticketing (`TICKETING_URL`)     |
| `/realms`, `/resources`, `/js` | Keycloak (`KEYCLOAK_URL`) — browser auth on one origin |
| everything else           | SPA static assets (`WEB_DIR`)   |

Proxying Keycloak lets the SPA, API, and login all share the `itsm.dev.mil`
origin. Backends then set `OIDC_DISCOVERY_URL` to the in-cluster Keycloak (so
discovery works without resolving the public host) while `OIDC_ISSUER` stays the
public `https://itsm.dev.mil/realms/...` that tokens carry. `/admin` is not
exposed.

When `WEB_DIR` is set the gateway serves the built SPA with history-API
fallback (unknown paths return `index.html`), so the browser talks to a single
origin for both the UI and the API. In dev, leave `WEB_DIR` empty and run the
SPA on the Vite dev server (which proxies `/api` back to the gateway).

## Configuration

| Env                | Notes                                                        |
|--------------------|--------------------------------------------------------------|
| `ADDR`             | Listen address, default `:8443`                              |
| `OIDC_ISSUER`      | When unset (dev), protected routes are disabled              |
| `OIDC_AUDIENCE`    | Expected `aud`                                               |
| `OIDC_DISCOVERY_URL` | Fetch OIDC metadata from here (in-cluster) while validating against `OIDC_ISSUER`; empty = discover at issuer |
| `KEYCLOAK_URL`     | In-cluster Keycloak to proxy `/realms`,`/resources`,`/js` to; empty disables |
| `TICKETING_URL`    | Ticketing upstream, e.g. `https://ticketing:8445`; empty disables |
| `INTERNAL_CA_FILE` | PEM CA for internal TLS; empty in dev = skip-verify          |
| `WEB_DIR`          | Built SPA assets dir to serve; empty disables (dev uses Vite) |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | Server cert; empty in dev = self-signed       |
