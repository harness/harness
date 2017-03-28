package sql

import (
	"github.com/drone/drone/store/datastore/sql/postgres"
	"github.com/drone/drone/store/datastore/sql/sqlite"
)

// Supported database drivers
const (
	DriverSqlite   = "sqlite3"
	DriverMysql    = "mysql"
	DriverPostgres = "postgres"
)

// Lookup returns the named sql statement compatible with
// the specified database driver.
func Lookup(driver string, name string) string {
	switch driver {
	case DriverPostgres:
		return postgres.Lookup(name)
	default:
		return sqlite.Lookup(name)
	}
}
