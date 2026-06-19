// Command identity is a minimal Phase 0 service that proves the shared /pkg
// conventions and the build/run/deploy path. It will grow into the user,
// group, and role projection service in Phase 1.
package main

import (
	"log/slog"

	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/gofiber/fiber/v2"
)

func main() { server.Bootstrap("identity", ":8444", run) }

func run(cfg config.Config, logger *slog.Logger) error {
	app := server.NewApp("usg-itsm-identity", nil)

	app.Get("/internal/v1/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"service": cfg.ServiceName, "status": "ok"})
	})

	return server.Run(cfg, app, logger)
}
