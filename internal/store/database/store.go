// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package database provides persistent data storage using
// a postgres or sqlite3 database.
package database

import (
	"database/sql"
	"time"

	"github.com/harness/gitness/internal/store/database/migrate"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// build is a global instance of the sql builder. we are able to
// hardcode to postgres since sqlite3 is compatible with postgres.
var builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// Connect to a database and verify with a ping.
func Connect(driver, datasource string) (*sqlx.DB, error) {
	db, err := sql.Open(driver, datasource)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open the db")
	}

	dbx := sqlx.NewDb(db, driver)
	if err = pingDatabase(dbx); err != nil {
		return nil, errors.Wrap(err, "Failed to ping the db")
	}

	if err = setupDatabase(dbx); err != nil {
		return nil, errors.Wrap(err, "Failed to setup the db")
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
func pingDatabase(db *sqlx.DB) error {
	var err error
	for i := 0; i < 30; i++ {
		err = db.Ping()

		// We can complete on first successful ping
		if err == nil {
			return nil
		}

		log.Err(err).Msgf("Ping attempt #%d failed", i+1)

		time.Sleep(time.Second)
	}

	return err
}

// helper function to setup the database by performing automated
// database migration steps.
func setupDatabase(db *sqlx.DB) error {
	return migrate.Migrate(db)
}
