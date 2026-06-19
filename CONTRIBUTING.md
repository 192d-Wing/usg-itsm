# Contributing to USG-ITSM

This repository follows a strict **branch → PR → review → merge** flow with
**Conventional Commits**. No direct commits to `main`.

## 1. Branch naming

Branch off the latest `main`. Use a type prefix that matches the change:

```
<type>/<short-kebab-summary>
```

| Prefix      | Use for                                            |
|-------------|----------------------------------------------------|
| `feat/`     | New capability                                     |
| `fix/`      | Bug fix                                             |
| `chore/`    | Tooling, deps, build, non-product changes          |
| `docs/`     | Documentation only                                 |
| `refactor/` | Behavior-preserving restructure                    |
| `test/`     | Tests only                                         |
| `ci/`       | CI/CD pipeline changes                             |

Example: `feat/ticketing-comments-api`

## 2. Conventional Commits

Every commit message **must** follow
[Conventional Commits 1.0.0](https://www.conventionalcommits.org/):

```
<type>(<optional scope>)<optional !>: <description>

[optional body]

[optional footer(s)]
```

Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
`build`, `ci`, `chore`, `revert`.

Scopes are the service or area, e.g. `gateway`, `ticketing`, `catalog`,
`workflow-sla`, `notification`, `audit`, `identity`, `pkg`, `web`, `deploy`.

Breaking changes: add `!` after the type/scope **and** a `BREAKING CHANGE:`
footer.

Examples:

```
feat(ticketing): add work-item state machine
fix(gateway): reject TLS < 1.3 on internal listener
chore(deps): bump fiber to v2.52
feat(audit)!: switch audit log to append-only hash chain

BREAKING CHANGE: audit records are no longer mutable; migration required.
```

### Enforcement

- **Locally**: run `make hooks` once after cloning to install the
  `commit-msg` hook (`.githooks/commit-msg`). It rejects non-conforming
  messages before they are created.
- **In CI**: the `commit-lint` workflow validates every commit on the PR and
  the PR title.

## 3. Pull Requests

1. Push your branch and open a PR against `main`.
2. The **PR title must itself be a valid Conventional Commit** — we squash-merge
   and the title becomes the commit on `main`.
3. Fill out the PR template (what/why/testing/risk).
4. CI must be green: build, vet, lint, unit tests, and commit-lint.
5. At least one approving review from a CODEOWNER is required.

## 4. Code review

- Every PR is reviewed before merge. Reviewers check correctness, security
  (this is a government/air-gapped system — no external runtime deps, TLS 1.3
  only, no secrets in code), tests, and adherence to the shared `/pkg`
  conventions.
- Authors run `/code-review` locally (or the repo's review tooling) before
  requesting review, and address findings.
- Merges to `main` are **squash-merge only**, keeping history linear and every
  commit on `main` conventional.

## 5. Local development

```bash
make hooks      # install git hooks (run once)
make up         # start the local stack (Postgres, NATS, Olric, SeaweedFS, Keycloak)
make build      # build all Go services
make test       # run unit tests
make lint       # go vet + gofmt check
make down       # stop the local stack
```

See [docs/architecture/overview.md](docs/architecture/overview.md) for the
system design and [docs/adr/](docs/adr/) for recorded decisions.
