package database

import (
	"database/sql"

	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/datastore/database/migrate"

	"github.com/BurntSushi/migration"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"
)

const (
	driverPostgres = "postgres"
	driverSqlite   = "sqlite3"
	driverMysql    = "mysql"
)

// Connect is a helper function that establishes a new
// database connection and auto-generates the database
// schema. If the database already exists, it will perform
// and update as needed.
func Connect(driver, datasource string) (*sql.DB, error) {
	switch driver {
	case driverPostgres:
		meddler.Default = meddler.PostgreSQL
	case driverSqlite:
		meddler.Default = meddler.SQLite
	case driverMysql:
		meddler.Default = meddler.MySQL
	}
	migration.DefaultGetVersion = migrate.GetVersion
	migration.DefaultSetVersion = migrate.SetVersion
	var migrations = []migration.Migrator{
		migrate.Setup,
	}
	return migration.Open(driver, datasource, migrations)
}

// MustConnect is a helper function that create a
// new database commention and auto-generates the
// database schema. An error causes a panic.
func MustConnect(driver, datasource string) *sql.DB {
	db, err := Connect(driver, datasource)
	if err != nil {
		panic(err)
	}
	return db
}

// New returns a new DataStore
func New(db *sql.DB) datastore.Datastore {
	return struct {
		*Userstore
		*Permstore
		*Repostore
		*Commitstore
	}{
		NewUserstore(db),
		NewPermstore(db),
		NewRepostore(db),
		NewCommitstore(db),
	}
}
