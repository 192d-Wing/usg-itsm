# 0007 — TLS 1.3 only + mTLS

- Status: Accepted
- Date: 2026-06-18

## Context

Government/air-gapped posture demands strong, modern transport security and a
minimal attack surface. Legacy TLS versions and cipher suites are liabilities.

## Decision

Enforce **TLS 1.3 only** at the ingress edge and on every service listener
(`MinVersion = tls.VersionTLS13`). Service-to-service traffic uses **mTLS**.

## Consequences

- TLS 1.3 removes all legacy ciphers and renegotiation, simplifying the FIPS /
  STIG hardening story.
- Older clients that cannot negotiate TLS 1.3 are rejected by design; this is
  acceptable for the controlled deployment environments we target.
- The shared `/pkg/tlsconf` package centralizes the TLS config so no service
  can accidentally weaken it.
- mTLS certificate issuance/rotation is handled by cert-manager with an internal
  CA in-cluster.
