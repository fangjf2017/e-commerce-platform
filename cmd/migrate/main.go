package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://fms:fmspass@localhost:5432/featureflags?sslmode=disable"
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer conn.Close(ctx)

	if err := ensureMigrationsTable(ctx, conn); err != nil {
		log.Fatalf("ensuring migrations table: %v", err)
	}

	migrationsDir := filepath.Join("internal", "db", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("reading migrations directory: %v", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	applied := 0
	for _, filename := range sqlFiles {
		ran, err := hasMigrationRun(ctx, conn, filename)
		if err != nil {
			log.Fatalf("checking migration %s: %v", filename, err)
		}
		if ran {
			log.Printf("skipping %s (already applied)", filename)
			continue
		}

		path := filepath.Join(migrationsDir, filename)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("reading migration file %s: %v", filename, err)
		}

		if err := runMigration(ctx, conn, filename, string(sqlBytes)); err != nil {
			log.Fatalf("running migration %s: %v", filename, err)
		}
		log.Printf("applied migration: %s", filename)
		applied++
	}

	if applied == 0 {
		log.Println("no new migrations to apply")
	} else {
		log.Printf("applied %d migration(s)", applied)
	}
}

func ensureMigrationsTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename    TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}
	return nil
}

func hasMigrationRun(ctx context.Context, conn *pgx.Conn, filename string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`,
		filename,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("querying schema_migrations: %w", err)
	}
	return exists, nil
}

func runMigration(ctx context.Context, conn *pgx.Conn, filename, sql string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("executing migration SQL: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (filename) VALUES ($1)`,
		filename,
	); err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	return tx.Commit(ctx)
}
