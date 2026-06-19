package store

import "embed"

// Migrations holds the embedded SQL migrations applied at startup via
// pkg/db.Migrate(ctx, pool, schema, Migrations, MigrationsDir).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the directory inside Migrations holding the .sql files.
const MigrationsDir = "migrations"
