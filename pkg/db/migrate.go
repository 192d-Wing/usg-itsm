package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate applies every *.sql file in dir of fsys, in lexical order, exactly
// once. Applied versions (the file names) are tracked in a schema_migrations
// table inside schema. Each migration runs in its own transaction, so a
// failure leaves earlier migrations applied and rolls back the failing one.
func Migrate(ctx context.Context, pool *pgxpool.Pool, schema string, fsys fs.FS, dir string) error {
	migTable := pgx.Identifier{schema, "schema_migrations"}.Sanitize()

	if _, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{schema}.Sanitize()); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := pool.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS "+migTable+
			" (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())",
	); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	versions, err := sqlVersions(fsys, dir)
	if err != nil {
		return err
	}

	for _, version := range versions {
		applied, err := isApplied(ctx, pool, migTable, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := fs.ReadFile(fsys, dir+"/"+version)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		if err := applyOne(ctx, pool, migTable, version, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
	}
	return nil
}

// sqlVersions returns the sorted .sql file names in dir of fsys.
func sqlVersions(fsys fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions)
	return versions, nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, migTable, version string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM "+migTable+" WHERE version = $1)", version,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return exists, nil
}

// applyOne runs a migration body and records its version atomically. The body
// is executed via the simple query protocol (PgConn().Exec), which — unlike
// pgx's default extended protocol — supports multiple statements in a single
// SQL string. The whole thing is wrapped in BEGIN/COMMIT so a failure rolls
// back, including the version insert.
func applyOne(ctx context.Context, pool *pgxpool.Pool, migTable, version, body string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// version is an embedded migration file name (trusted), but double any
	// quote defensively before inlining it into the simple-protocol batch.
	safeVersion := strings.ReplaceAll(version, "'", "''")
	batch := "BEGIN;\n" + body +
		"\nINSERT INTO " + migTable + " (version) VALUES ('" + safeVersion + "');\nCOMMIT;"

	_, err = conn.Conn().PgConn().Exec(ctx, batch).ReadAll()
	return err
}
