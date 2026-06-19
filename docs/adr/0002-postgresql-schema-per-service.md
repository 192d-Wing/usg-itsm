# 0002 — PostgreSQL, schema-per-service

- Status: Accepted
- Date: 2026-06-18

## Context

Each service must own its data, but operating six separate database clusters in
an air-gapped environment for an MVP is excessive overhead.

## Decision

Use a single **PostgreSQL** (HA) cluster with **one schema per service**.
Services connect with a role scoped to their own schema and never read another
service's tables — integration happens only via APIs and emitted events.

## Consequences

- Logical isolation now, a clean seam to split into separate clusters later.
- PostgreSQL is the **single source of truth**; caches and search indexes are
  derived and disposable.
- Cross-service queries are forbidden; reporting uses event projections or a
  dedicated read model, not direct table joins.
