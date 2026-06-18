# 0006 — SeaweedFS for object storage

- Status: Accepted
- Date: 2026-06-18

## Context

Ticket attachments need S3-compatible object storage. MinIO's open-source
edition was gutted in 2025 (console and features moved behind the commercial
license), so it is no longer a sound OSS foundation.

## Decision

Use **SeaweedFS** (Apache-2.0) for S3-compatible object storage.

## Consequences

- Apache-2.0 — the cleanest license posture for government use.
- Optimized for large numbers of small files, matching attachment workloads.
- Small footprint, S3 API, fully self-hostable and mirror-able for air-gap.
- Alternatives rejected: Garage (AGPLv3); Ceph RGW (enterprise-grade but heavy,
  justified only if Ceph is already operated for block/file storage).
