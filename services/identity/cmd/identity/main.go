// Command identity is a minimal Phase 0 service that proves the shared /pkg
// conventions and the build/run/deploy path. It will grow into the user,
// group, and role projection service in Phase 1.
package main

import (
	"log/slog"
	"os"

	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/pkg/log"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	cfg := config.Load("identity", ":8444")
	logger := log.New(cfg.ServiceName, cfg.LogLevel)

	if err := run(cfg, logger); err != nil {
		logger.Error("identity exited with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	app := fiber.New(fiber.Config{
		AppName:               "usg-itsm-identity",
		DisableStartupMessage: true,
		ErrorHandler:          httpx.DefaultErrorHandler,
	})
	app.Use(requestid.New())
	app.Use(recover.New())

	httpx.Health(app, nil)

	app.Get("/internal/v1/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"service": cfg.ServiceName, "status": "ok"})
	})

	return server.Run(cfg, app, logger)
}
