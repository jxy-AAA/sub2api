package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type healthMigrationChecker interface {
	Check(ctx context.Context, db *sql.DB) error
}

type routeHealthChecker interface {
	Check(ctx context.Context) error
}

type healthDBProvider interface {
	RawDB() (*sql.DB, error)
}

type migrationReadinessChecker struct{}

func (migrationReadinessChecker) Check(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("db not configured")
	}

	var hasSchema bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)
	`).Scan(&hasSchema); err != nil {
		return fmt.Errorf("check schema_migrations: %w", err)
	}
	if !hasSchema {
		return errors.New("schema_migrations missing")
	}

	var hasAtlas bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'atlas_schema_revisions'
		)
	`).Scan(&hasAtlas); err != nil {
		return fmt.Errorf("check atlas_schema_revisions: %w", err)
	}
	if !hasAtlas {
		return errors.New("atlas_schema_revisions missing")
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM atlas_schema_revisions").Scan(&count); err != nil {
		return fmt.Errorf("count atlas_schema_revisions: %w", err)
	}
	if count <= 0 {
		return errors.New("atlas_schema_revisions empty")
	}

	return nil
}

type postgresReadinessChecker struct {
	source           healthDBProvider
	pingTimeout      time.Duration
	migrationChecker healthMigrationChecker
}

func (c postgresReadinessChecker) Check(ctx context.Context) error {
	if c.source == nil {
		return errors.New("postgres source not configured")
	}
	db, err := c.source.RawDB()
	if err != nil {
		return err
	}
	if db == nil {
		return errors.New("postgres db not configured")
	}

	timeout := c.pingTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	checker := c.migrationChecker
	if checker == nil {
		checker = migrationReadinessChecker{}
	}
	if err := checker.Check(ctx, db); err != nil {
		return fmt.Errorf("check migrations: %w", err)
	}
	return nil
}
