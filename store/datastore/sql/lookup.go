package sql

import (
	"github.com/drone/drone/store/datastore/sql/sqlite"
)

// Supported database drivers
const (
	DriverSqlite   = "sqlite"
	DriverMysql    = "mysql"
	DriverPostgres = "postgres"
)

// Lookup returns the named sql statement compatible with
// the specified database driver.
func Lookup(driver string, name string) string {
	switch driver {
	default:
		return sqlite.Lookup(name)
	}
}
