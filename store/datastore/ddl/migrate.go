package ddl

import (
	"database/sql"
	"errors"

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
	if err := checkPriorMigration(db); err != nil {
		return err
	}
	switch driver {
	case DriverMysql:
		return mysql.Migrate(db)
	case DriverPostgres:
		return postgres.Migrate(db)
	default:
		return sqlite.Migrate(db)
	}
}

// we need to check and see if there was a previous migration
// for drone 0.6 or prior and migrate to the new migration
// system. Attempting to migrate from 0.5 or below to 0.7 or
// above will result in an error.
//
// this can be removed once we get to 1.0 with the reasonable
// expectation that people are no longer using 0.5.
func checkPriorMigration(db *sql.DB) error {
	var none int
	if err := db.QueryRow(legacyMigrationsExist).Scan(&none); err != nil {
		// if no legacy migrations exist, this is a fresh install
		// and we can proceed as normal.
		return nil
	}
	if err := db.QueryRow(legacyMigrationsCurrent).Scan(&none); err != nil {
		// this indicates an attempted upgrade from 0.5 or lower to
		// version 0.7 or higher and will fail.
		return errors.New("Please upgrade to 0.6 before upgrading to 0.7+")
	}
	db.Exec(createMigrationsTable)
	db.Exec(legacyMigrationsImport)
	return nil
}

var legacyMigrationsExist = `
SELECT 1
FROM gorp_migrations
LIMIT 1
`

var legacyMigrationsCurrent = `
SELECT 1
FROM gorp_migrations
WHERE id = '16.sql'
LIMIT 1
`

var legacyMigrationsImport = `
INSERT INTO migrations (name) VALUES
 ('create-table-users')
,('create-table-repos')
,('create-table-builds')
,('create-index-builds-repo')
,('create-index-builds-author')
,('create-table-procs')
,('create-index-procs-build')
,('create-table-logs')
,('create-table-files')
,('create-index-files-builds')
,('create-index-files-procs')
,('create-table-secrets')
,('create-index-secrets-repo')
,('create-table-registry')
,('create-index-registry-repo')
,('create-table-config')
,('create-table-tasks')
,('create-table-agents')
,('create-table-senders')
,('create-index-sender-repos')
`

var createMigrationsTable = `
CREATE TABLE migrations (
 name VARCHAR(255)
,UNIQUE(name)
)
`
