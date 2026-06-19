package gw

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

// WebUI serves the built SPA from dir with history-API fallback: unknown paths
// (client-side routes like /tickets/:id) return index.html so the SPA router
// can handle them. API and health paths are skipped so they fall through to
// their own handlers. Register this AFTER the API/health routes.
func WebUI(dir string) fiber.Handler {
	return filesystem.New(filesystem.Config{
		Root:         http.Dir(dir),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Next: func(c *fiber.Ctx) bool {
			p := c.Path()
			return strings.HasPrefix(p, "/api/") || p == "/healthz" || p == "/readyz"
		},
	})
}
