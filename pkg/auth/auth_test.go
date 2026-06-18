package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// fakeVerifier returns canned claims or an error, with no network dependency.
type fakeVerifier struct {
	claims *Claims
	err    error
}

func (f fakeVerifier) Verify(_ context.Context, _ string) (*Claims, error) {
	return f.claims, f.err
}

func newApp(v Verifier) *fiber.App {
	app := fiber.New()
	app.Get("/me", RequireAuth(v), func(c *fiber.Ctx) error {
		return c.JSON(From(c))
	})
	app.Get("/admin", RequireAuth(v), RequireRole("admin"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func do(t *testing.T, app *fiber.App, path, authHeader string) int {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	if authHeader != "" {
		req.Header.Set(fiber.HeaderAuthorization, authHeader)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp.StatusCode
}

func TestRequireAuth_MissingToken(t *testing.T) {
	app := newApp(fakeVerifier{claims: &Claims{Subject: "u1"}})
	if got := do(t, app, "/me", ""); got != fiber.StatusUnauthorized {
		t.Fatalf("want 401, got %d", got)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	app := newApp(fakeVerifier{err: errors.New("bad")})
	if got := do(t, app, "/me", "Bearer xxx"); got != fiber.StatusUnauthorized {
		t.Fatalf("want 401, got %d", got)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	app := newApp(fakeVerifier{claims: &Claims{Subject: "u1", Roles: []string{"agent"}}})
	if got := do(t, app, "/me", "Bearer good"); got != fiber.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	app := newApp(fakeVerifier{claims: &Claims{Subject: "u1", Roles: []string{"agent"}}})
	if got := do(t, app, "/admin", "Bearer good"); got != fiber.StatusForbidden {
		t.Fatalf("want 403, got %d", got)
	}
}

func TestRequireRole_Allowed(t *testing.T) {
	app := newApp(fakeVerifier{claims: &Claims{Subject: "u1", Roles: []string{"admin"}}})
	if got := do(t, app, "/admin", "Bearer good"); got != fiber.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

func TestExtractRoles_RealmAccessFallback(t *testing.T) {
	raw := map[string]any{
		"realm_access": map[string]any{
			"roles": []any{"agent", "approver"},
		},
	}
	roles := extractRoles(raw, "roles")
	if len(roles) != 2 || roles[0] != "agent" || roles[1] != "approver" {
		t.Fatalf("unexpected roles: %v", roles)
	}
}
