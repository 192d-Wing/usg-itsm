# 0010 — Gateway BFF routing to backend services

- Status: Accepted
- Date: 2026-06-19

## Context

The gateway is the only ingress (TLS edge). Browser clients must reach backend
services (ticketing, and later catalog/workflow/etc.) through it, not directly.
Backend services listen on TLS only and validate OIDC tokens themselves.

## Decision

The gateway acts as a **BFF/reverse proxy**. It validates the OIDC bearer token
at the `/api/v1` group, then forwards matching paths to the relevant upstream,
preserving method, path, query, body, and headers — **including `Authorization`,
so the upstream re-validates the token (defense in depth)**.

- Routing uses Fiber's native `proxy` (fasthttp) with a per-call TLS 1.3 client
  (`tlsconf.Client`) so internal hops stay TLS 1.3 (ADR-0007).
- Upstreams are configured by URL (e.g. `TICKETING_URL`); an unset upstream
  disables that route. A failed upstream call returns `502`.
- Internal certificate trust comes from `INTERNAL_CA_FILE` (the cert-manager
  internal CA in-cluster). In dev, with no CA, verification is skipped.

## Consequences

- One enforcement point for ingress auth; services keep independent validation.
- `/api/v1/tickets` and subpaths now route to the ticketing service.
- Adding a service is a config + two route lines; no new ingress surface.
- The fasthttp proxy keeps the hot path native (no net/http conversion). If
  richer rewriting is needed later, revisit per-route handlers.
- The browser only ever talks to the gateway over the edge LB (IPv4 VIP →
  IPv6 backend, ADR-0009).
