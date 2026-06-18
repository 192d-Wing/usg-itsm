# Web (SPA)

The USG-ITSM single-page app (Phase 1+). Planned stack:

- **React 18 + TypeScript + Vite**
- **Tailwind CSS + shadcn/ui** (Radix primitives), self-hosted fonts/icons —
  no external CDN, so it works air-gapped and meets Section 508 / WCAG.
- **TanStack Query** for server state, **React Router**, and
  **react-hook-form + zod** for dynamic catalog forms.

Two surfaces: a self-service **portal** (submit/track requests, knowledge base)
and an **agent workspace** (queues, ticket detail, SLA timers).

Built to static assets and served by the gateway; authenticates against the
configured OIDC provider using the `usg-itsm-web` public client.
