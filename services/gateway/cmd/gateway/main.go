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
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/192d-Wing/usg-itsm/pkg/tlsconf"
	"github.com/192d-Wing/usg-itsm/services/gateway/internal/gw"
	"github.com/gofiber/fiber/v2"
)

func main() { server.Bootstrap("gateway", ":8443", run) }

func run(cfg config.Config, logger *slog.Logger) error {
	app := server.NewApp("usg-itsm-gateway", nil)

	// Runtime SPA config: the served bundle is environment-agnostic and fetches
	// its OIDC settings here at startup, so one image works across deployments.
	app.Get("/config.json", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"oidcAuthority": cfg.OIDCIssuer,
			"oidcClientId":  cfg.OIDCAudience,
		})
	})

	// Build the OIDC verifier when configured. In dev without an issuer the
	// gateway still serves public routes but protected routes are unavailable.
	api := app.Group("/api/v1")
	if cfg.AuthEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		verifier, err := auth.NewOIDCVerifier(ctx, cfg.OIDCIssuer, cfg.OIDCDiscoveryURL, cfg.OIDCAudience, cfg.RolesClaim)
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

	// Proxy Keycloak's browser-facing paths so the SPA, API, and auth all share
	// the itsm.dev.mil origin. Mounted before the SPA so it isn't shadowed.
	if err := mountKeycloak(app, cfg, logger); err != nil {
		return err
	}

	// Serve the built SPA (with history-API fallback) when configured. Mounted
	// last so it never shadows the API or health routes.
	if cfg.WebDir != "" {
		app.Use(gw.WebUI(cfg.WebDir))
		logger.Info("serving SPA", "dir", cfg.WebDir)
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

// mountKeycloak proxies Keycloak's browser-facing paths so login happens on the
// single itsm.dev.mil origin. /admin is intentionally not exposed.
func mountKeycloak(app *fiber.App, cfg config.Config, logger *slog.Logger) error {
	if cfg.KeycloakURL == "" {
		return nil
	}
	if u, err := url.Parse(cfg.KeycloakURL); err != nil ||
		(u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid KEYCLOAK_URL %q: must be an http(s) URL", cfg.KeycloakURL)
	}
	clientTLS, err := tlsconf.Client(cfg.InternalCAFile, cfg.IsDev())
	if err != nil {
		return err
	}
	h := gw.Proxy(cfg.KeycloakURL, gw.NewUpstreamClient(clientTLS), gw.DefaultUpstreamTimeout)
	for _, p := range []string{"/realms", "/resources", "/js"} {
		app.All(p, h)
		app.All(p+"/*", h)
	}
	logger.Info("routing Keycloak auth paths -> keycloak", "upstream", cfg.KeycloakURL)
	return nil
}

// me returns the validated caller's claims.
func me(c *fiber.Ctx) error {
	return c.JSON(auth.From(c))
}
