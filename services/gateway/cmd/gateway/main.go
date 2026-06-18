// Command gateway is the TLS edge / BFF for USG-ITSM.
//
// It terminates TLS 1.3, validates OIDC bearer tokens, enforces coarse RBAC,
// and (in later phases) routes to backend services and serves the SPA.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/pkg/log"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	cfg := config.Load("gateway", ":8443")
	logger := log.New(cfg.ServiceName, cfg.LogLevel)

	if err := run(cfg, logger); err != nil {
		logger.Error("gateway exited with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	app := fiber.New(fiber.Config{
		AppName:               "usg-itsm-gateway",
		DisableStartupMessage: true,
		ErrorHandler:          httpx.DefaultErrorHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())

	httpx.Health(app, nil)

	// Build the OIDC verifier when configured. In dev without an issuer the
	// gateway still serves public routes but protected routes are unavailable.
	api := app.Group("/api/v1")
	if cfg.AuthEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		verifier, err := auth.NewOIDCVerifier(ctx, cfg.OIDCIssuer, cfg.OIDCAudience, cfg.RolesClaim)
		if err != nil {
			return err
		}
		api.Get("/me", auth.RequireAuth(verifier), me)
		logger.Info("OIDC auth enabled", "issuer", cfg.OIDCIssuer)
	} else {
		logger.Warn("OIDC issuer not set; protected API routes are disabled (dev only)")
	}

	return server.Run(cfg, app, logger)
}

// me returns the validated caller's claims.
func me(c *fiber.Ctx) error {
	return c.JSON(auth.From(c))
}
