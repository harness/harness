// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package dbtest

import (
	"os"

	"github.com/drone/drone/store/shared/db"

	// blank imports are used to load database drivers
	// for unit tests. Only unit tests should be importing
	// this package.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a new test database connection.
func Connect() (*db.DB, error) {
	var (
		driver = "sqlite3"
		config = ":memory:?_foreign_keys=1"
	)
	if os.Getenv("DATABASE_DRIVER") != "" {
		driver = os.Getenv("DATABASE_DRIVER")
		config = os.Getenv("DATABASE_CONFIG")
	}
	return db.Connect(driver, config)
}

// Reset resets the database state.
func Reset(d *db.DB) {
	d.Lock(func(tx db.Execer, _ db.Binder) error {
		tx.Exec("DELETE FROM cron")
		tx.Exec("DELETE FROM logs")
		tx.Exec("DELETE FROM steps")
		tx.Exec("DELETE FROM stages")
		tx.Exec("DELETE FROM builds")
		tx.Exec("DELETE FROM perms")
		tx.Exec("DELETE FROM repos")
		tx.Exec("DELETE FROM users")
		return nil
	})
}

// Disconnect closes the database connection.
func Disconnect(d *db.DB) error {
	return d.Close()
}
