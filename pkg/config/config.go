// Package config loads service configuration from the environment.
//
// All services use the same loader so configuration is consistent and
// air-gap friendly (no config server, just environment variables / mounted
// secrets).
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the common configuration shared by every service.
type Config struct {
	// ServiceName identifies the service in logs and traces.
	ServiceName string

	// Addr is the listen address, e.g. ":8443".
	Addr string

	// Environment is "dev", "staging", or "prod".
	Environment string

	// LogLevel is one of debug, info, warn, error.
	LogLevel string

	// TLSCertFile / TLSKeyFile point at the PEM cert/key. When empty in a
	// dev environment, services generate an in-memory self-signed cert.
	TLSCertFile string
	TLSKeyFile  string

	// OIDC settings. Issuer empty disables auth enforcement (dev only).
	OIDCIssuer   string
	OIDCAudience string
	// RolesClaim is the JWT claim holding role names. Defaults handle both a
	// top-level array claim and Keycloak's realm_access.roles.
	RolesClaim string

	// DatabaseURL is the PostgreSQL connection string (empty if the service is
	// stateless). DatabaseSchema scopes the connection's search_path so each
	// service owns its schema (ADR-0002).
	DatabaseURL    string
	DatabaseSchema string

	// NatsURL is the NATS JetStream endpoint for event publishing. Empty
	// disables publishing (events still persist in the database).
	NatsURL string

	// TicketingURL is the ticketing upstream the gateway routes to (gateway
	// only), e.g. https://ticketing:8445. Empty disables ticket routing.
	TicketingURL string

	// InternalCAFile is the PEM CA bundle used to verify internal service
	// certificates for service-to-service TLS. Empty in dev falls back to
	// skip-verify; outside dev a CA (or system trust) is used.
	InternalCAFile string

	// WebDir is the directory of built SPA assets the gateway serves (gateway
	// only). Empty disables static serving (dev runs the SPA on Vite).
	WebDir string

	// ShutdownTimeout bounds graceful shutdown.
	ShutdownTimeout time.Duration
}

// Load builds a Config for the named service from environment variables.
// defaultAddr is the service's listen address when ADDR is not set.
func Load(service, defaultAddr string) Config {
	return Config{
		ServiceName:     service,
		Addr:            env("ADDR", defaultAddr),
		Environment:     env("ENVIRONMENT", "dev"),
		LogLevel:        env("LOG_LEVEL", "info"),
		TLSCertFile:     env("TLS_CERT_FILE", ""),
		TLSKeyFile:      env("TLS_KEY_FILE", ""),
		OIDCIssuer:      env("OIDC_ISSUER", ""),
		OIDCAudience:    env("OIDC_AUDIENCE", ""),
		RolesClaim:      env("OIDC_ROLES_CLAIM", "roles"),
		DatabaseURL:     env("DATABASE_URL", ""),
		DatabaseSchema:  env("DATABASE_SCHEMA", service),
		NatsURL:         env("NATS_URL", ""),
		TicketingURL:    env("TICKETING_URL", ""),
		InternalCAFile:  env("INTERNAL_CA_FILE", ""),
		WebDir:          env("WEB_DIR", ""),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}
}

// IsDev reports whether the service runs in a development environment.
func (c Config) IsDev() bool {
	return strings.EqualFold(c.Environment, "dev")
}

// AuthEnabled reports whether OIDC token validation is enforced.
func (c Config) AuthEnabled() bool {
	return c.OIDCIssuer != ""
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}
