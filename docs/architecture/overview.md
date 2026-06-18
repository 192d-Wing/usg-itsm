# USG-ITSM — Architecture Overview

## Goals & constraints

- **Air-gapped Kubernetes** target. No external SaaS, CDN, or public package
  dependency at runtime. Every image and asset must be mirror-able into a
  private registry (Harbor).
- **Permissive licensing.** Apache-2.0 / BSD components preferred; AGPL avoided
  in the data path. Recorded per-component in [ADRs](../adr/).
- **Security-first.** TLS 1.3 only, mTLS between services, OIDC-agnostic auth,
  RBAC, and an append-only audit trail.
- **Sleek, modern UX** for both self-service requesters and agents.

## System context

```
                     ┌──────────────────────────────────────────┐
   Browser ────────▶ │  Ingress (nginx) + cert-manager (TLS 1.3) │
                     └────────────────────┬─────────────────────┘
                                          │
                          ┌───────────────▼───────────────┐
                          │   API Gateway / BFF (Fiber)    │
                          │   JWKS/JWT validate, RBAC,     │
                          │   routing, rate-limit, SPA host│
                          └───┬───────┬───────┬───────┬────┘
                              │       │       │       │
                ┌─────────────┘  ┌────▼───┐   │   ┌───▼─────────┐
          ┌─────▼────┐  ┌────────▼──┐ │ Ticketing │ │ Workflow / │
          │ Identity │  │  Catalog  │ │  (work    │ │ SLA engine │
          │ /User    │  │ (Svc Req) │ │  items)   │ │            │
          └─────┬────┘  └─────┬─────┘ └─────┬─────┘ └─────┬──────┘
                │             │             │             │
                └─────────────┴─ NATS JetStream (events + KV) ─┴──┐
                                          │                       │
                              ┌───────────┴──────┐         ┌──────▼─────┐
                              │   Notification    │         │   Audit    │
                              │   (email/webhook) │         │   (log)    │
                              └───────────────────┘         └────────────┘

   Shared infra: PostgreSQL (schema-per-service) · Olric (cache) ·
                 SeaweedFS (S3 attachments) · OpenSearch (Phase 3)
```

## Services

| Service        | Responsibility                                              | State |
|----------------|-------------------------------------------------------------|-------|
| `gateway`      | TLS edge, OIDC/JWT validation, RBAC, routing, serves the SPA | stateless |
| `identity`     | Projects users/groups/roles from OIDC claims; assignment groups | Postgres |
| `ticketing`    | Work items (incidents + service requests), comments, attachment metadata | Postgres + SeaweedFS |
| `catalog`      | Service-request catalog items and dynamic form definitions  | Postgres |
| `workflow-sla` | State machines, approvals, SLA timers, escalations          | Postgres + NATS KV |
| `notification` | Templated email (SMTP relay) + webhook fan-out, event-driven | stateless |
| `audit`        | Append-only, tamper-evident audit trail                     | Postgres |

### Why incidents and service requests share one service

Both are *work items* (the ServiceNow `task` pattern): they share lifecycle,
assignment, SLA, comments, and audit. A single `ticketing` service with a
`type` discriminator avoids duplicating that engine. They diverge only in
forms, catalog entry points, and SLA policy.

## Cross-cutting concerns

- **Shared library `/pkg`** gives every service identical config loading,
  structured logging (`slog`), OIDC middleware, TLS 1.3 setup, OpenTelemetry
  wiring, NATS client, Olric client, and a uniform HTTP error envelope.
- **Auth**: services trust signed JWTs validated against the IdP's JWKS. No
  binding to a specific IdP — see [ADR-0005](../adr/0005-oidc-agnostic-auth.md).
- **AuthZ**: RBAC roles (`requester`, `agent`, `approver`, `admin`) plus
  assignment-group scoping, enforced at the gateway and re-checked per service.
- **Events**: domain events (`ticket.created`, `sla.breached`, …) flow over
  NATS JetStream; durable shared state that must survive a cache wipe uses
  NATS KV, never Olric alone.
- **Observability**: OTel traces/metrics/logs → Prometheus + Loki + Tempo,
  visualized in Grafana.

## Data ownership

Each service owns its schema in a shared PostgreSQL cluster (logical isolation
for the MVP, a clean seam to split into separate clusters later). Services
never read each other's tables — only their APIs and emitted events.

## Phased roadmap

- **Phase 0 — Foundation** *(current)*: monorepo, `/pkg`, local stack, gateway +
  identity vertical slice with OIDC.
- **Phase 1 — Ticketing core**: work items, comments, attachments, audit, agent UI.
- **Phase 2 — Workflow & SLA**: state machine, routing, SLA timers, notifications, approvals.
- **Phase 3 — Catalog & portal**: dynamic forms, self-service portal, OpenSearch.
- **Phase 4 — Hardening**: RBAC depth, 508/WCAG, observability dashboards,
  Helm/Argo packaging, air-gap image bundle + install runbook.
