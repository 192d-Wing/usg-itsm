// Command ticketing is the work-item service: incidents and service requests,
// their comments, and an append-only history (ADR-0008). It owns the
// `ticketing` schema and validates OIDC bearer tokens independently of the
// gateway (defense in depth).
package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/db"
	"github.com/192d-Wing/usg-itsm/pkg/events"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/api"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/store"
)

func main() { server.Bootstrap("ticketing", ":8445", run) }

func run(cfg config.Config, logger *slog.Logger) error {
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if !cfg.AuthEnabled() {
		return errors.New("OIDC_ISSUER is required")
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Connect(startupCtx, cfg.DatabaseURL, cfg.DatabaseSchema)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(startupCtx, pool, cfg.DatabaseSchema, store.Migrations, store.MigrationsDir); err != nil {
		return err
	}
	logger.Info("migrations applied", "schema", cfg.DatabaseSchema)

	verifier, err := auth.NewOIDCVerifier(startupCtx, cfg.OIDCIssuer, cfg.OIDCAudience, cfg.RolesClaim)
	if err != nil {
		return err
	}

	// Event publishing is optional: without NATS, events still persist to the
	// database; with it, ticket changes fan out on "itsm.ticket.*" (ADR-0004).
	storeOpts := []store.Option{}
	if cfg.NatsURL != "" {
		pub, err := events.Connect(startupCtx, cfg.NatsURL, "ITSM", []string{"itsm.>"})
		if err != nil {
			return err
		}
		defer pub.Close()
		storeOpts = append(storeOpts, store.WithPublisher(pub))
		logger.Info("event publishing enabled", "nats", cfg.NatsURL)
	} else {
		logger.Warn("NATS_URL not set; ticket events persist to the database only")
	}

	handlers := api.New(store.New(pool, storeOpts...))

	app := server.NewApp("usg-itsm-ticketing", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return pool.Ping(ctx)
	})

	v1 := app.Group("/api/v1", auth.RequireAuth(verifier))
	handlers.Register(v1)
	logger.Info("ticketing API mounted", "issuer", cfg.OIDCIssuer)

	return server.Run(cfg, app, logger)
}
