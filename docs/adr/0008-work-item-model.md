# 0008 — Incidents + service requests as work items

- Status: Accepted
- Date: 2026-06-18

## Context

The MVP covers incident management and service-request fulfillment. Both share
roughly 80% of their domain: state lifecycle, assignment, SLA, comments, audit.

## Decision

Model both as a single **work item** in the `ticketing` service, discriminated
by a `type` field (`incident` | `service_request`). This mirrors the
ServiceNow `task` superclass pattern.

## Consequences

- One state machine, assignment engine, comment/attachment subsystem, and audit
  path — no duplication.
- Types diverge only where they must: distinct intake forms, catalog entry
  points (service requests originate from the catalog), and SLA policies.
- Future work-item types (problem, change) extend the same core rather than
  introducing parallel services.
