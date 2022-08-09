// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package migrate

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/maragudk/migrate"
	"github.com/rs/zerolog/log"
)

// background context
var noContext = context.Background()

//go:embed postgres/*.sql
var postgres embed.FS

//go:embed sqlite/*.sql
var sqlite embed.FS

// Migrate performs the database migration.
func Migrate(db *sqlx.DB) error {
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
		Table:  "migrations",
	}

	switch db.DriverName() {
	case "postgres":
		folder, _ := fs.Sub(postgres, "postgres")
		opts.FS = folder

	default:
		folder, _ := fs.Sub(sqlite, "sqlite")
		opts.FS = folder
	}

	return migrate.New(opts).MigrateUp(noContext)
}
