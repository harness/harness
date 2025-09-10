// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package database provides persistent data storage using
// a postgres or sqlite3 database.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const (
	// sqlForUpdate is the sql statement used for locking rows returned by select queries.
	SQLForUpdate = "FOR UPDATE"
)

type Migrator func(ctx context.Context, dbx *sqlx.DB) error

// Builder is a global instance of the sql builder. we are able to
// hardcode to postgres since sqlite3 is compatible with postgres.
var Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// Connect to a database and verify with a ping.
func Connect(ctx context.Context, driver string, datasource string) (*sqlx.DB, error) {
	datasource, err := prepareDatasourceForDriver(driver, datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare datasource: %w", err)
	}

	db, err := sql.Open(driver, datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to open the db: %w", err)
	}

	dbx := sqlx.NewDb(db, driver)
	if err = pingDatabase(ctx, dbx); err != nil {
		return nil, fmt.Errorf("failed to ping the db: %w", err)
	}

	log.Ctx(ctx).Info().Str("driver", driver).Msg("Database connected")

	return dbx, nil
}

// ConnectAndMigrate creates the database handle and migrates the database.
func ConnectAndMigrate(
	ctx context.Context,
	driver string,
	datasource string,
	migrators ...Migrator,
) (*sqlx.DB, error) {
	dbx, err := Connect(ctx, driver, datasource)
	if err != nil {
		return nil, err
	}

	for _, migrator := range migrators {
		if err = migrator(ctx, dbx); err != nil {
			return nil, fmt.Errorf("failed to setup the db: %w", err)
		}
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

// prepareDatasourceForDriver ensures that required features are enabled on the
// datasource connection string based on the driver.
func prepareDatasourceForDriver(driver string, datasource string) (string, error) {
	switch driver {
	case "sqlite3":
		url, err := url.Parse(datasource)
		if err != nil {
			return "", fmt.Errorf("datasource is of invalid format for driver sqlite3")
		}

		// get original query and update it with required settings
		query := url.Query()

		// ensure foreign keys are always enabled (disabled by default)
		// See https://github.com/mattn/go-sqlite3#connection-string
		query.Set("_foreign_keys", "on")

		// update url with updated query
		url.RawQuery = query.Encode()

		return url.String(), nil
	default:
		return datasource, nil
	}
}

// helper function to ping the database with backoff to ensure
// a connection can be established before we proceed with the
// database setup and migration.
func pingDatabase(ctx context.Context, db *sqlx.DB) error {
	var err error
	for i := 1; i <= 30; i++ {
		err = db.PingContext(ctx)

		// No point in continuing if context was cancelled
		if errors.Is(err, context.Canceled) {
			return err
		}

		// We can complete on first successful ping
		if err == nil {
			return nil
		}

		log.Debug().Err(err).Msgf("Ping attempt #%d failed", i)

		time.Sleep(time.Second)
	}

	return fmt.Errorf("all 30 tries failed, last failure: %w", err)
}
