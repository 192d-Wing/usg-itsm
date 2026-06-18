# ticketing

The work-item service: **incidents** and **service requests** share one model
(ADR-0008), with comments and an append-only history. Owns the `ticketing`
PostgreSQL schema and validates OIDC bearer tokens independently of the gateway.

## API (under the authenticated `/api/v1` group)

| Method | Path                          | Role        | Description                          |
|--------|-------------------------------|-------------|--------------------------------------|
| POST   | `/tickets`                    | any         | Open a work item                     |
| GET    | `/tickets`                    | any         | List (requesters see only their own) |
| GET    | `/tickets/:id`                | owner/agent | Fetch one                            |
| PATCH  | `/tickets/:id`                | agent       | Update fields / assign               |
| POST   | `/tickets/:id/transition`     | agent       | Change status (state-machine checked)|
| GET    | `/tickets/:id/comments`       | owner/agent | List comments (agents see internal)  |
| POST   | `/tickets/:id/comments`       | owner/agent | Add a comment (internal = agent only)|
| GET    | `/tickets/:id/events`         | agent       | History / audit trail                |

Lifecycle: `new → in_progress ⇄ on_hold → resolved → closed`, with `cancelled`
reachable from any non-terminal state and `resolved → in_progress` to reopen.

## Configuration

| Env              | Required | Notes                                            |
|------------------|----------|--------------------------------------------------|
| `DATABASE_URL`   | yes      | PostgreSQL DSN                                    |
| `DATABASE_SCHEMA`| no       | Defaults to `ticketing`                          |
| `OIDC_ISSUER`    | yes      | Token validation issuer                          |
| `OIDC_AUDIENCE`  | yes      | Expected `aud`                                   |
| `ADDR`           | no       | Defaults to `:8445`                              |

Migrations in `internal/store/migrations/` are embedded and applied at startup.

## Tests

- `internal/domain` — pure state-machine and helper unit tests (always run).
- `internal/api` — handler tests with a fake store + fake verifier (always run).
- `internal/store` — integration tests; run only when `TEST_DATABASE_URL` is set
  (CI provides a Postgres service).
