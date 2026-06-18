// Command identity is a minimal Phase 0 service that proves the shared /pkg
// conventions and the build/run/deploy path. It will grow into the user,
// group, and role projection service in Phase 1.
package main

import (
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/pkg/log"
	"github.com/192d-Wing/usg-itsm/pkg/tlsconf"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	cfg := config.Load("identity")
	// identity listens internally on 8444 by default.
	if cfg.Addr == ":8443" {
		cfg.Addr = ":8444"
	}
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

	var tlsCfg *tls.Config
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		var err error
		if tlsCfg, err = tlsconf.LoadServer(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			return err
		}
	} else {
		if !cfg.IsDev() {
			return errors.New("TLS_CERT_FILE and TLS_KEY_FILE are required outside dev")
		}
		logger.Warn("no TLS cert configured; generating an in-memory self-signed cert (dev only)")
		cert, err := tlsconf.SelfSigned("localhost")
		if err != nil {
			return err
		}
		tlsCfg = tlsconf.Server(cert)
	}

	ln, err := tls.Listen("tcp", cfg.Addr, tlsCfg)
	if err != nil {
		return err
	}

	go func() {
		logger.Info("identity listening", "addr", cfg.Addr, "tls", "1.3-only")
		if err := app.Listener(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error("listener stopped", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("shutdown signal received")
	return app.ShutdownWithTimeout(cfg.ShutdownTimeout)
}
