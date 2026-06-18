# 0005 — OIDC-agnostic authentication

- Status: Accepted
- Date: 2026-06-18

## Context

Different operators use different identity providers (Keycloak, Okta, Entra ID)
and government deployments may require PIV/CAC smart-card auth. The platform
must not hard-bind to one IdP.

## Decision

Services authenticate by validating **signed JWTs** against the configured
provider's **JWKS** endpoint, using standard OIDC discovery. Any compliant
OIDC/OAuth2 provider works via configuration alone (issuer URL, audience,
JWKS). No provider-specific code in services.

## Consequences

- The gateway validates tokens (signature, `iss`, `aud`, `exp`) and forwards
  identity/roles downstream; services re-verify for defense in depth.
- PIV/CAC integration is handled at the IdP (e.g. Keycloak x.509 client-cert
  auth), keeping smart-card complexity out of the application tier.
- Local development ships a Keycloak realm; production points at the operator's
  IdP with no code change.
