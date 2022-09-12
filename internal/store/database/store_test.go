// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"os"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var noContext = context.Background()

// connect opens a new test database connection.
func connect() (*sqlx.DB, error) {
	var (
		driver = "sqlite3"
		config = ":memory:?_foreign_keys=1"
	)
	if os.Getenv("DATABASE_CONFIG") != "" {
		driver = os.Getenv("DATABASE_DRIVER")
		config = os.Getenv("DATABASE_CONFIG")
	}
	return Connect(driver, config)
}

// seed seed the database state.
func seed(db *sqlx.DB) error {
	_, _ = db.Exec("DELETE FROM executions")
	_, _ = db.Exec("DELETE FROM pipelines")
	_, _ = db.Exec("DELETE FROM users")
	_, _ = db.Exec("ALTER SEQUENCE users_user_id_seq RESTART WITH 1")
	_, _ = db.Exec("ALTER SEQUENCE pipelines_pipeline_id_seq RESTART WITH 1")
	_, _ = db.Exec("ALTER SEQUENCE executions_execution_id_seq RESTART WITH 1")
	return nil
}

// unmarshal a testdata file.
func unmarshal(path string, v interface{}) error {
	out, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, v)
}
