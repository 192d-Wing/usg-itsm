package gw_test

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/192d-Wing/usg-itsm/services/gateway/internal/gw"
	"github.com/gofiber/fiber/v2"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func webApp(t *testing.T) *fiber.App {
	t.Helper()
	// Not t.TempDir(): on Windows its strict cleanup fails when the static
	// handler still holds an open file handle. Best-effort removal instead.
	dir, err := os.MkdirTemp("", "webui")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	writeFile(t, filepath.Join(dir, "index.html"), "<!doctype html><title>spa</title>")
	writeFile(t, filepath.Join(dir, "assets", "app.js"), "console.log('app')")

	app := fiber.New()
	// API + health are registered before the SPA fallback.
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/api/v1/ping", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"pong": true}) })
	app.Use(gw.WebUI(dir))
	return app
}

func get(t *testing.T, app *fiber.App, path string) (int, string) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("test %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestWebUI_ServesIndex(t *testing.T) {
	app := webApp(t)
	status, body := get(t, app, "/")
	if status != fiber.StatusOK || !strings.Contains(body, "spa") {
		t.Fatalf("index: status=%d body=%q", status, body)
	}
}

func TestWebUI_ServesAsset(t *testing.T) {
	app := webApp(t)
	status, body := get(t, app, "/assets/app.js")
	if status != fiber.StatusOK || !strings.Contains(body, "console.log") {
		t.Fatalf("asset: status=%d body=%q", status, body)
	}
}

func TestWebUI_SPAFallback(t *testing.T) {
	app := webApp(t)
	// Unknown client-side route falls back to index.html.
	status, body := get(t, app, "/tickets/123")
	if status != fiber.StatusOK || !strings.Contains(body, "spa") {
		t.Fatalf("fallback: status=%d body=%q", status, body)
	}
}

func TestWebUI_DoesNotShadowAPI(t *testing.T) {
	app := webApp(t)
	status, body := get(t, app, "/api/v1/ping")
	if status != fiber.StatusOK || !strings.Contains(body, "pong") {
		t.Fatalf("api: status=%d body=%q", status, body)
	}
}

func TestWebUI_DoesNotShadowHealth(t *testing.T) {
	app := webApp(t)
	status, body := get(t, app, "/healthz")
	if status != fiber.StatusOK || !strings.Contains(body, "ok") {
		t.Fatalf("health: status=%d body=%q", status, body)
	}
}
