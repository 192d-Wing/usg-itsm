# 0003 — Olric (Apache-2.0) for caching

- Status: Accepted
- Date: 2026-06-18

## Context

We need a distributed cache. Redis 8 is **AGPLv3**; Valkey/KeyDB/Memcached are
BSD — none are Apache-2.0, which the project requires for the cache layer.

## Decision

Use **Olric** (Apache-2.0), a Go-native distributed cache, in client-server
mode. It covers hot lookups, rate-limit counters, ephemeral SLA-timer state,
and distributed locks.

## Consequences

- License requirement satisfied; Go-native client matches the stack and the
  server mirrors into Harbor as a small static binary.
- **Maturity risk**: Olric has a smaller community than Redis. Mitigations:
  - PostgreSQL remains the source of truth — nothing lives *only* in Olric.
  - Durable shared state that must survive a cache wipe uses **NATS KV**
    ([ADR-0004](0004-nats-jetstream-events.md)), not Olric.
  - Cache loss degrades to "cold cache, slightly slower," never data loss.
- Alternatives rejected: Apache Ignite / Hazelcast (Apache-2.0 but heavyweight
  JVM data grids, fat containers).
