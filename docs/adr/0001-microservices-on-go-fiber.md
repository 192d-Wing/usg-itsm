# 0001 — Microservices on Go + Fiber

- Status: Accepted
- Date: 2026-06-18

## Context

We need small, fast, statically-linked services that produce minimal container
images suitable for high-density, air-gapped Kubernetes. The team chose Go.

## Decision

Build each service in **Go**, using **Fiber** as the HTTP framework. Services
compile to a single static binary on a `distroless`/`scratch` base.

## Consequences

- Fiber runs on **fasthttp**, not the stdlib `net/http`. Implications:
  - OIDC/JWT validation uses pure token libraries (`coreos/go-oidc`,
    `golang-jwt`) — no transport coupling, so this is unaffected.
  - Tracing uses the Fiber-native **`otelfiber`** middleware, not the
    `net/http` instrumentation.
  - Any third-party middleware must be Fiber-native or wrapped via Fiber's
    `adaptor`. The shared `/pkg` is Fiber-first so each service stays consistent.
- Tiny images, fast cold start, low memory — ideal for air-gapped clusters.
