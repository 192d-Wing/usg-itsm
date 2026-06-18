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
	"github.com/192d-Wing/usg-itsm/pkg/tlsconf"
	"github.com/gofiber/fiber/v2"
)

// Run serves app on a TLS 1.3 listener bound to cfg.Addr and blocks until a
// shutdown signal (SIGINT/SIGTERM) or a fatal listener error. It returns the
// first error encountered, so a listener that dies causes the process to exit
// rather than hang.
func Run(cfg config.Config, app *fiber.App, logger *slog.Logger) error {
	tlsCfg, err := serverTLS(cfg, logger)
	if err != nil {
		return err
	}

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
