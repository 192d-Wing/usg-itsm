# web — agent workspace SPA

The USG-ITSM single-page app. Phase 1 delivers the **agent workspace**: a ticket
queue, ticket detail (comments, status transitions, assignment), and a new-ticket
form, talking to `/api/v1/tickets` through the gateway.

## Stack

- **React 18 + TypeScript + Vite**
- **Tailwind CSS** with a small in-repo component set (Button, Badge, Card,
  fields, Spinner) — self-hosted, no external CDN (air-gap friendly, 508/WCAG).
- **TanStack Query** for server state, **React Router** for routing.
- **OIDC (Auth Code + PKCE)** via `react-oidc-context` / `oidc-client-ts` —
  IdP-agnostic; configured by `VITE_OIDC_AUTHORITY`.

## Develop

```bash
cp .env.example .env.local   # adjust OIDC authority / client if needed
npm install
npm run dev                  # http://localhost:5173
```

The dev server proxies `/api` to the gateway at `https://localhost:8443`
(self-signed cert accepted in dev). Run the backend stack (`make up`) and the
gateway + ticketing services so the SPA has an API and an OIDC issuer.

## Scripts

| Script              | Purpose                          |
|---------------------|----------------------------------|
| `npm run dev`       | Vite dev server                  |
| `npm run build`     | Type-check + production build     |
| `npm run typecheck` | `tsc` only                       |
| `npm run lint`      | ESLint                           |

## Layout

- `src/api/` — typed fetch client + ticketing endpoints
- `src/auth/` — OIDC config
- `src/features/tickets/` — queue, detail, new-ticket screens + query hooks
- `src/components/ui/` — design-system primitives
- `src/components/layout/` — app shell (sidebar/topbar)

Roles come from the access token (`roles` / Keycloak `realm_access.roles`):
agents get transition/assignment/internal-note controls; requesters see their
own tickets. The gateway and ticketing service enforce authorization
server-side regardless.
