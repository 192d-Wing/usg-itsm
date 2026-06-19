// Package server provides the shared HTTP server runner used by every service:
// a TLS 1.3-only listener (ADR-0007), a dev self-signed fallback, and graceful
// shutdown that also exits if the listener fails fatally.
package server

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

// Bootstrap is the standard main() for a service: load config for the named
// service, build the logger, run run, and exit non-zero on error.
func Bootstrap(service, defaultAddr string, run func(config.Config, *slog.Logger) error) {
	cfg := config.Load(service, defaultAddr)
	logger := log.New(cfg.ServiceName, cfg.LogLevel)
	if err := run(cfg, logger); err != nil {
		logger.Error(service+" exited with error", "err", err)
		os.Exit(1)
	}
}

// NewApp builds a Fiber app with the standard middleware stack every service
// shares: the uniform error envelope, request IDs, panic recovery, and the
// liveness/readiness endpoints. ready may be nil. Mount service routes on the
// returned app, then pass it to Run.
func NewApp(name string, ready func() error) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               name,
		DisableStartupMessage: true,
		ErrorHandler:          httpx.DefaultErrorHandler,
	})
	app.Use(requestid.New())
	app.Use(recover.New())
	httpx.Health(app, ready)
	return app
}

// Run serves app on a TLS 1.3 listener bound to cfg.Addr and blocks until a
// shutdown signal (SIGINT/SIGTERM) or a fatal listener error. It returns the
// first error encountered, so a listener that dies causes the process to exit
// rather than hang.
func Run(cfg config.Config, app *fiber.App, logger *slog.Logger) error {
	tlsCfg, err := serverTLS(cfg, logger)
	if err != nil {
		return err
	}

	// A ":<port>" address binds [::] (dual-stack), which is correct on the
	// IPv6-only cluster network (ADR-0009); IPv4 is terminated at the edge LB.
	ln, err := tls.Listen("tcp", cfg.Addr, tlsCfg)
	if err != nil {
		return err
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "tls", "1.3-only")
		if err := app.Listener(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			serveErr <- err
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		logger.Error("listener failed", "err", err)
		return err
	case s := <-sig:
		logger.Info("shutdown signal received",
			"signal", s.String(), "timeout", cfg.ShutdownTimeout.String())
		return app.ShutdownWithTimeout(cfg.ShutdownTimeout)
	}
}

// serverTLS builds a TLS 1.3 config, generating an in-memory self-signed cert
// when no cert files are configured in a dev environment.
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
