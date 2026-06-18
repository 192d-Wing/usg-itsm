# 0004 — NATS JetStream for events + KV

- Status: Accepted
- Date: 2026-06-18

## Context

Services communicate asynchronously via domain events and occasionally need
durable shared key-value state. Kafka is powerful but heavy to operate in an
air-gapped cluster.

## Decision

Use **NATS JetStream** as the event backbone and **NATS KV** for durable shared
state.

## Consequences

- A single small Go binary, far lighter than a Kafka + ZooKeeper/KRaft stack,
  and trivially mirror-able into Harbor.
- JetStream provides at-least-once delivery, replay, and consumer durability —
  sufficient for `ticket.created`, `sla.breached`, `comment.added`, etc.
- NATS KV backs durable cross-service state (e.g. SLA timer checkpoints) so the
  cache ([ADR-0003](0003-olric-cache.md)) never holds the only copy.
- If future throughput demands Kafka semantics, the event interface in `/pkg`
  abstracts the broker, limiting the blast radius of a swap.
