// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package database provides persistent data storage using
// a postgres or sqlite3 database.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/store/database/migrate"
	"github.com/rs/zerolog/log"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// build is a global instance of the sql builder. we are able to
// hardcode to postgres since sqlite3 is compatible with postgres.
var builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// Connect to a database and verify with a ping.
func Connect(ctx context.Context, driver string, datasource string) (*sqlx.DB, error) {
	db, err := sql.Open(driver, datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to open the db: %w", err)
	}

	dbx := sqlx.NewDb(db, driver)
	if err = pingDatabase(ctx, dbx); err != nil {
		return nil, fmt.Errorf("failed to ping the db: %w", err)
	}

	if err = setupDatabase(ctx, dbx); err != nil {
		return nil, fmt.Errorf("failed to setup the db: %w", err)
	}

	return dbx, nil
}

// Must is a helper function that wraps a call to Connect
// and panics if the error is non-nil.
func Must(db *sqlx.DB, err error) *sqlx.DB {
	if err != nil {
		panic(err)
	}
	return db
}

// helper function to ping the database with backoff to ensure
// a connection can be established before we proceed with the
// database setup and migration.
func pingDatabase(ctx context.Context, db *sqlx.DB) error {
	var err error
	for i := 1; i <= 30; i++ {
		err = db.PingContext(ctx)

		// We can complete on first successful ping
		if err == nil {
			return nil
		}

		log.Debug().Err(err).Msgf("Ping attempt #%d failed", i)

		time.Sleep(time.Second)
	}

	return fmt.Errorf("all 30 tries failed, last failure: %w", err)
}

// helper function to setup the database by performing automated
// database migration steps.
func setupDatabase(ctx context.Context, db *sqlx.DB) error {
	return migrate.Migrate(ctx, db)
}
