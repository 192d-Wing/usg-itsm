// Package auth provides OIDC-agnostic token validation and Fiber middleware.
//
// Per ADR-0005, services trust signed JWTs validated against the configured
// provider's JWKS. No provider-specific code lives here — any compliant
// OIDC/OAuth2 issuer works through configuration alone.
package auth

import (
	"context"
	"slices"
	"strings"

	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/gofiber/fiber/v2"
)

// localsKey is the Fiber Locals key under which verified claims are stored.
const localsKey = "auth.claims"

// Claims is the normalized identity extracted from a validated token.
type Claims struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	Roles   []string `json:"roles"`
}

// HasRole reports whether the subject holds the named role.
func (c *Claims) HasRole(role string) bool {
	return slices.Contains(c.Roles, role)
}

// Verifier validates a raw bearer token and returns normalized claims.
// It is an interface so services can inject fakes in tests.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}

// RequireAuth returns middleware that rejects requests without a valid bearer
// token and stores the resulting claims in Fiber Locals.
func RequireAuth(v Verifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := bearerToken(c)
		if raw == "" {
			return httpx.WriteError(c, fiber.StatusUnauthorized, "unauthorized",
				"missing bearer token")
		}
		claims, err := v.Verify(c.UserContext(), raw)
		if err != nil {
			return httpx.WriteError(c, fiber.StatusUnauthorized, "unauthorized",
				"invalid or expired token")
		}
		c.Locals(localsKey, claims)
		return c.Next()
	}
}

// RequireRole returns middleware that enforces a role. It must run after
// RequireAuth.
func RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := From(c)
		if claims == nil {
			return httpx.WriteError(c, fiber.StatusUnauthorized, "unauthorized",
				"authentication required")
		}
		if !claims.HasRole(role) {
			return httpx.WriteError(c, fiber.StatusForbidden, "forbidden",
				"insufficient role")
		}
		return c.Next()
	}
}

// From returns the validated claims stored on the request, or nil.
func From(c *fiber.Ctx) *Claims {
	if claims, ok := c.Locals(localsKey).(*Claims); ok {
		return claims
	}
	return nil
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header. Per RFC 7235 the auth scheme is matched case-insensitively.
func bearerToken(c *fiber.Ctx) string {
	const prefix = "bearer "
	h := c.Get(fiber.HeaderAuthorization)
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
