// Command gateway is the TLS edge / BFF for USG-ITSM.
//
// It terminates TLS 1.3, validates OIDC bearer tokens, enforces coarse RBAC,
// and (in later phases) routes to backend services and serves the SPA.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/pkg/log"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/192d-Wing/usg-itsm/pkg/tlsconf"
	"github.com/192d-Wing/usg-itsm/services/gateway/internal/gw"
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
		api.Use(auth.RequireAuth(verifier))
		api.Get("/me", me)
		logger.Info("OIDC auth enabled", "issuer", cfg.OIDCIssuer)

		if err := mountUpstreams(api, cfg, logger); err != nil {
			return err
		}
	} else {
		logger.Warn("OIDC issuer not set; protected API routes are disabled (dev only)")
	}

	return server.Run(cfg, app, logger)
}

// mountUpstreams wires gateway routing to backend services over TLS 1.3.
func mountUpstreams(r fiber.Router, cfg config.Config, logger *slog.Logger) error {
	if cfg.TicketingURL == "" {
		logger.Warn("TICKETING_URL not set; ticket routing disabled")
		return nil
	}
	if u, err := url.Parse(cfg.TicketingURL); err != nil ||
		(u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid TICKETING_URL %q: must be an http(s) URL", cfg.TicketingURL)
	}
	// Internal certs are issued by the cluster CA (cert-manager), which is not
	// in the system trust store; require it outside dev so a missing CA fails
	// fast instead of surfacing as opaque 502s.
	if !cfg.IsDev() && cfg.InternalCAFile == "" {
		return errors.New("INTERNAL_CA_FILE is required outside dev for internal TLS verification")
	}

	clientTLS, err := tlsconf.Client(cfg.InternalCAFile, cfg.IsDev())
	if err != nil {
		return err
	}
	h := gw.Proxy(cfg.TicketingURL, gw.NewUpstreamClient(clientTLS), gw.DefaultUpstreamTimeout)
	r.All("/tickets", h)
	r.All("/tickets/*", h)
	logger.Info("routing /api/v1/tickets -> ticketing",
		"upstream", cfg.TicketingURL, "insecure_skip_verify", cfg.InternalCAFile == "" && cfg.IsDev())
	return nil
}

// me returns the validated caller's claims.
func me(c *fiber.Ctx) error {
	return c.JSON(auth.From(c))
}
