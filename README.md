# USG-ITSM

A containerized, microservices ITSM web platform — a modern, self-hostable
alternative to ServiceNow / BMC Remedy, built for **air-gapped Kubernetes**.

> Status: **Phase 0 — Foundation.** Monorepo, shared libraries, local stack,
> and an OIDC-authenticated vertical slice through the gateway.

## Why

Government and air-gapped operators need ITSM without external SaaS
dependencies, with strong audit, modern UX, and a permissive-license stack.

## Stack

| Concern         | Choice                          | Notes                                   |
|-----------------|---------------------------------|-----------------------------------------|
| Language        | **Go** + **Fiber**              | Tiny, statically-linked containers       |
| Datastore       | **PostgreSQL** (schema-per-svc) | Single source of truth                   |
| Cache           | **Olric** (Apache-2.0)          | Go-native distributed cache              |
| Events / KV     | **NATS JetStream**              | Lightweight event bus + durable KV       |
| Object storage  | **SeaweedFS** (Apache-2.0)      | S3-compatible attachment store           |
| Search          | **OpenSearch** (Phase 3)        | Ticket / KB full-text                    |
| Identity        | **OIDC-agnostic**               | Keycloak, Okta, Entra — any OIDC/OAuth2  |
| Frontend        | **React + TS + Vite + Tailwind**| shadcn/ui, self-hosted assets            |
| Orchestration   | **Kubernetes + Helm + Argo CD** | Harbor registry, fully mirror-able       |
| Transport       | **TLS 1.3 only** + mTLS         | No legacy ciphers                        |

See [docs/architecture/overview.md](docs/architecture/overview.md) for the
full design and [docs/adr/](docs/adr/) for the recorded decisions.

## Services (MVP)

| Service          | Responsibility                                            |
|------------------|-----------------------------------------------------------|
| `gateway`        | TLS edge, OIDC/JWT validation, RBAC, routing, serves SPA  |
| `identity`       | User/group/role projection from OIDC claims               |
| `ticketing`      | Work items: incidents **and** service requests            |
| `catalog`        | Service-request catalog items + dynamic forms             |
| `workflow-sla`   | State machines, approvals, SLA timers, escalations        |
| `notification`   | Email + webhook fan-out, event-driven                     |
| `audit`          | Append-only, tamper-evident audit trail                   |

## Quick start (local dev)

Requires Docker, Go 1.25+, and `make`.

```bash
make hooks      # install the Conventional Commits git hook (run once)
make up         # start Postgres, NATS, Olric, SeaweedFS, Keycloak
make run-gateway
```

Then hit the gateway health endpoint:

```bash
curl -k https://localhost:8443/healthz
```

## Contributing

All work happens on branches with a PR and code review. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/). See
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE).
