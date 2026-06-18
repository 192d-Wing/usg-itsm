// Command gateway is the TLS edge / BFF for USG-ITSM.
//
// It terminates TLS 1.3, validates OIDC bearer tokens, enforces coarse RBAC,
// and (in later phases) routes to backend services and serves the SPA.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/pkg/log"
	"github.com/192d-Wing/usg-itsm/pkg/tlsconf"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	cfg := config.Load("gateway")
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

	tlsCfg, err := serverTLS(cfg, logger)
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", cfg.Addr, tlsCfg)
	if err != nil {
		return err
	}

	go func() {
		logger.Info("gateway listening", "addr", cfg.Addr, "tls", "1.3-only")
		if err := app.Listener(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error("listener stopped", "err", err)
		}
	}()

	return waitForShutdown(app, cfg, logger)
}

// me returns the validated caller's claims.
func me(c *fiber.Ctx) error {
	return c.JSON(auth.From(c))
}

// serverTLS builds a TLS 1.3 config, generating a dev self-signed cert when no
// cert files are configured in a dev environment.
func serverTLS(cfg config.Config, logger *slog.Logger) (*tls.Config, error) {
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		return tlsconf.LoadServer(cfg.TLSCertFile, cfg.TLSKeyFile)
	}
	if !cfg.IsDev() {
		return nil, errors.New("TLS_CERT_FILE and TLS_KEY_FILE are required outside dev")
	}
	logger.Warn("no TLS cert configured; generating an in-memory self-signed cert (dev only)")
	cert, err := tlsconf.SelfSigned("localhost")
	if err != nil {
		return nil, err
	}
	return tlsconf.Server(cert), nil
}

func waitForShutdown(app *fiber.App, cfg config.Config, logger *slog.Logger) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("shutdown signal received", "timeout", cfg.ShutdownTimeout.String())
	return app.ShutdownWithTimeout(cfg.ShutdownTimeout)
}
