package ddl

import (
	"database/sql"

	"github.com/drone/drone/store/datastore/ddl/mysql"
	"github.com/drone/drone/store/datastore/ddl/postgres"
	"github.com/drone/drone/store/datastore/ddl/sqlite"
)

// Supported database drivers
const (
	DriverSqlite   = "sqlite3"
	DriverMysql    = "mysql"
	DriverPostgres = "postgres"
)

// Migrate performs the database migration. If the migration fails
// and error is returned.
func Migrate(driver string, db *sql.DB) error {
	switch driver {
	case DriverMysql:
		return mysql.Migrate(db)
	case DriverPostgres:
		return postgres.Migrate(db)
	default:
		return sqlite.Migrate(db)
	}
}
