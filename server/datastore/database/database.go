package database

import (
	"database/sql"
	"os"

	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/datastore/migrate"

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
		migrate.Migrate_20142110,
		migrate.Migrate_20152701,
	}
	return migration.Open(driver, datasource, migrations)
}

// MustConnect is a helper function that creates a
// new database connection and auto-generates the
// database schema. An error causes a panic.
func MustConnect(driver, datasource string) *sql.DB {
	db, err := Connect(driver, datasource)
	if err != nil {
		panic(err)
	}
	return db
}

// mustConnectTest is a helper function that creates a
// new database connection using environment variables.
// If not environment varaibles are found, the default
// in-memory SQLite database is used.
func mustConnectTest() *sql.DB {
	var (
		driver     = os.Getenv("TEST_DRIVER")
		datasource = os.Getenv("TEST_DATASOURCE")
	)
	if len(driver) == 0 {
		driver = driverSqlite
		datasource = ":memory:"
	}
	db, err := Connect(driver, datasource)
	if err != nil {
		panic(err)
	}
	return db
}

// New returns a new Datastore
func NewDatastore(db *sql.DB) datastore.Datastore {
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
