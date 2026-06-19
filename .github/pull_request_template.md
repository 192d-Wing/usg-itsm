<!--
PR title MUST be a valid Conventional Commit, e.g.:
  feat(ticketing): add work-item state machine
It becomes the squash-merge commit on main.
-->

## What

<!-- What does this change do? -->

## Why

<!-- Motivation / linked issue. -->

## How it was tested

<!-- Commands run, manual steps, screenshots for UI. -->

## Risk & rollout

<!-- Migrations? Breaking changes? Air-gap/compliance impact? -->

## Checklist

- [ ] PR title is a valid Conventional Commit
- [ ] Branch follows `type/short-summary` naming
- [ ] `make build` and `make test` pass locally
- [ ] No secrets, credentials, or external runtime dependencies introduced
- [ ] TLS 1.3 / mTLS preserved for any new network surface
- [ ] Docs/ADR updated if a decision or interface changed
- [ ] Ran `/code-review` and addressed findings
