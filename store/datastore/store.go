package datastore

import (
	"database/sql"
	"os"
	"time"

	"github.com/drone/drone/store"
	"github.com/drone/drone/store/datastore/ddl"
	"github.com/russross/meddler"

	"github.com/Sirupsen/logrus"
)

// datastore is an implementation of a model.Store built on top
// of the sql/database driver with a relational database backend.
type datastore struct {
	*sql.DB

	driver string
	config string
}

// New creates a database connection for the given driver and datasource
// and returns a new Store.
func New(driver, config string) store.Store {
	return &datastore{
		DB:     open(driver, config),
		driver: driver,
		config: config,
	}
}

// From returns a Store using an existing database connection.
func From(db *sql.DB) store.Store {
	return &datastore{DB: db}
}

// open opens a new database connection with the specified
// driver and connection string and returns a store.
func open(driver, config string) *sql.DB {
	db, err := sql.Open(driver, config)
	if err != nil {
		logrus.Errorln(err)
		logrus.Fatalln("database connection failed")
	}
	if driver == "mysql" {
		// per issue https://github.com/go-sql-driver/mysql/issues/257
		db.SetMaxIdleConns(0)
	}

	setupMeddler(driver)

	if err := pingDatabase(db); err != nil {
		logrus.Errorln(err)
		logrus.Fatalln("database ping attempts failed")
	}

	if err := setupDatabase(driver, db); err != nil {
		logrus.Errorln(err)
		logrus.Fatalln("migration failed")
	}
	return db
}

// openTest opens a new database connection for testing purposes.
// The database driver and connection string are provided by
// environment variables, with fallback to in-memory sqlite.
func openTest() *sql.DB {
	var (
		driver = "sqlite3"
		config = ":memory:"
	)
	if os.Getenv("DATABASE_DRIVER") != "" {
		driver = os.Getenv("DATABASE_DRIVER")
		config = os.Getenv("DATABASE_CONFIG")
	}
	return open(driver, config)
}

// newTest creates a new database connection for testing purposes.
// The database driver and connection string are provided by
// environment variables, with fallback to in-memory sqlite.
func newTest() *datastore {
	var (
		driver = "sqlite3"
		config = ":memory:"
	)
	if os.Getenv("DATABASE_DRIVER") != "" {
		driver = os.Getenv("DATABASE_DRIVER")
		config = os.Getenv("DATABASE_CONFIG")
	}
	return &datastore{
		DB:     open(driver, config),
		driver: driver,
		config: config,
	}
}

// helper function to ping the database with backoff to ensure
// a connection can be established before we proceed with the
// database setup and migration.
func pingDatabase(db *sql.DB) (err error) {
	for i := 0; i < 30; i++ {
		err = db.Ping()
		if err == nil {
			return
		}
		logrus.Infof("database ping failed. retry in 1s")
		time.Sleep(time.Second)
	}
	return
}

// helper function to setup the databsae by performing
// automated database migration steps.
func setupDatabase(driver string, db *sql.DB) error {
	return ddl.Migrate(driver, db)
}

// helper function to setup the meddler default driver
// based on the selected driver name.
func setupMeddler(driver string) {
	switch driver {
	case "sqlite3":
		meddler.Default = meddler.SQLite
	case "mysql":
		meddler.Default = meddler.MySQL
	case "postgres":
		meddler.Default = meddler.PostgreSQL
	}
}
