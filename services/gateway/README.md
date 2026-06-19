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
| `GET /api/v1/me`          | gateway (returns caller claims) |
| `/api/v1/tickets[/*]`     | ticketing (`TICKETING_URL`)     |

## Configuration

| Env                | Notes                                                        |
|--------------------|--------------------------------------------------------------|
| `ADDR`             | Listen address, default `:8443`                              |
| `OIDC_ISSUER`      | When unset (dev), protected routes are disabled              |
| `OIDC_AUDIENCE`    | Expected `aud`                                               |
| `TICKETING_URL`    | Ticketing upstream, e.g. `https://ticketing:8445`; empty disables |
| `INTERNAL_CA_FILE` | PEM CA for internal TLS; empty in dev = skip-verify          |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | Server cert; empty in dev = self-signed       |
