package testdatabase

import (
	"database/sql"
	"os"

	// database drivers that may be tested
	_ "github.com/mattn/go-sqlite3"
)

var (
	driver = env("TEST_DB_DRIVER", "sqlite3")
	source = env("TEST_DB_SOURCE", ":memory:")
)

// Open opens a new database connection using a test
// database environment, specified using the `$TEST_DB_DRIVER`
// and `$TEST_DB_SOURCE` environment variables.
func Open() (*sql.DB, error) {
	return sql.Open(driver, source)
}

// helper function that retrieves the environment variable
// if exists, else returns a default value.
func env(name, def string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		value = def
	}

	return value
}
