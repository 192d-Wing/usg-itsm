package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCVerifier validates JWTs against an OIDC provider's JWKS using standard
// discovery. It implements Verifier.
type OIDCVerifier struct {
	verifier   *oidc.IDTokenVerifier
	rolesClaim string
}

// NewOIDCVerifier performs OIDC discovery against issuer and returns a verifier
// bound to the given audience. rolesClaim names the top-level claim holding
// role names; Keycloak's realm_access.roles is also checked as a fallback.
func NewOIDCVerifier(ctx context.Context, issuer, audience, rolesClaim string) (*OIDCVerifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	if rolesClaim == "" {
		rolesClaim = "roles"
	}
	return &OIDCVerifier{
		verifier:   provider.Verifier(&oidc.Config{ClientID: audience}),
		rolesClaim: rolesClaim,
	}, nil
}

// Verify validates the token signature, issuer, audience, and expiry, then
// extracts normalized claims.
func (o *OIDCVerifier) Verify(ctx context.Context, rawToken string) (*Claims, error) {
	tok, err := o.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	var raw map[string]any
	if err := tok.Claims(&raw); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	return &Claims{
		Subject: tok.Subject,
		Email:   stringClaim(raw, "email"),
		Name:    stringClaim(raw, "name"),
		Roles:   extractRoles(raw, o.rolesClaim),
	}, nil
}

func stringClaim(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// extractRoles reads roles from the configured claim, falling back to
// Keycloak's realm_access.roles structure.
func extractRoles(m map[string]any, rolesClaim string) []string {
	if roles := toStringSlice(m[rolesClaim]); len(roles) > 0 {
		return roles
	}
	if ra, ok := m["realm_access"].(map[string]any); ok {
		return toStringSlice(ra["roles"])
	}
	return nil
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
