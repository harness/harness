// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/maragudk/migrate"
	"github.com/rs/zerolog/log"
)

//go:embed postgres/*.sql
var postgres embed.FS

//go:embed sqlite/*.sql
var sqlite embed.FS

const tableName = "migrations"

// Migrate performs the database migration.
func Migrate(ctx context.Context, db *sqlx.DB) error {
	opts := getMigrator(db)
	return migrate.New(opts).MigrateUp(ctx)
}

// To performs the database migration to the specific version.
func To(ctx context.Context, db *sqlx.DB, version string) error {
	opts := getMigrator(db)
	return migrate.New(opts).MigrateTo(ctx, version)
}

// Current returns the current version ID (the latest migration applied) of the database.
func Current(ctx context.Context, db *sqlx.DB) (string, error) {
	var (
		query               string
		migrationTableCount int
	)

	switch db.DriverName() {
	case "sqlite3":
		query = `
			SELECT count(*)
			FROM sqlite_master
			WHERE name = ? and type = 'table'`
	default:
		query = `
			SELECT count(*)
			FROM information_schema.tables
			WHERE table_name = ? and table_schema = 'public'`
	}

	if err := db.QueryRowContext(ctx, query, tableName).Scan(&migrationTableCount); err != nil {
		return "", fmt.Errorf("failed to check migration table existence: %w", err)
	}

	if migrationTableCount == 0 {
		return "", nil
	}

	var version string

	query = "select version from " + tableName + " limit 1"
	if err := db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		return "", fmt.Errorf("failed to read current DB version from migration table: %w", err)
	}

	return version, nil
}

func getMigrator(db *sqlx.DB) migrate.Options {
	before := func(_ context.Context, _ *sql.Tx, version string) error {
		log.Trace().Str("version", version).Msg("migration started")
		return nil
	}

	after := func(_ context.Context, _ *sql.Tx, version string) error {
		log.Trace().Str("version", version).Msg("migration complete")
		return nil
	}

	opts := migrate.Options{
		After:  after,
		Before: before,
		DB:     db.DB,
		FS:     sqlite,
		Table:  tableName,
	}

	switch db.DriverName() {
	case "postgres":
		folder, _ := fs.Sub(postgres, "postgres")
		opts.FS = folder

	default:
		folder, _ := fs.Sub(sqlite, "sqlite")
		opts.FS = folder
	}

	return opts
}
