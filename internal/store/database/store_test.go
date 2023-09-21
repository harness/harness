// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"context"
	"encoding/json"
	"os"

	"github.com/harness/gitness/store/database"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// connect opens a new test database connection.
func connect() (*sqlx.DB, error) {
	var (
		driver = "sqlite3"
		config = ":memory:"
	)
	if os.Getenv("DATABASE_CONFIG") != "" {
		driver = os.Getenv("DATABASE_DRIVER")
		config = os.Getenv("DATABASE_CONFIG")
	}
	return database.Connect(context.Background(), driver, config)
}

// seed seed the database state.
func seed(db *sqlx.DB) error {
	/*
		_, err := db.Exec("DELETE FROM executions")
		if err != nil {
			return err
		}
		_, err = db.Exec("DELETE FROM pipelines")
		if err != nil {
			return err
		}
		_, err = db.Exec("DELETE FROM users")
		if err != nil {
			return err
		}
		_, err = db.Exec("ALTER SEQUENCE users_user_id_seq RESTART WITH 1")
		if err != nil {
			return err
		}
		_, err = db.Exec("ALTER SEQUENCE pipelines_pipeline_id_seq RESTART WITH 1")
		if err != nil {
			return err
		}
		_, err = db.Exec("ALTER SEQUENCE executions_execution_id_seq RESTART WITH 1")
		return err
	*/
	return nil
}

// unmarshal a testdata file.
//
//nolint:unparam // expected to be called for other paths in the future.
func unmarshal(path string, v interface{}) error {
	out, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, v)
}
