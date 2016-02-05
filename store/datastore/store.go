package datastore

import (
	"database/sql"
	"os"
	"time"

	"github.com/CiscoCloud/drone/shared/envconfig"
	"github.com/CiscoCloud/drone/store"
	"github.com/CiscoCloud/drone/store/migration"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rubenv/sql-migrate"
	"github.com/russross/meddler"

	log "github.com/Sirupsen/logrus"
)

// Load opens a new database connection with the specified driver
// and connection string specified in the environment variables.
func Load(env envconfig.Env) store.Store {
	var (
		driver = env.String("DATABASE_DRIVER", "sqlite3")
		config = env.String("DATABASE_CONFIG", "drone.sqlite")
	)

	log.Infof("using database driver %s", driver)
	log.Infof("using database config %s", config)

	return New(driver, config)
}

func New(driver, config string) store.Store {
	db := Open(driver, config)
	return store.New(
		driver,
		&nodestore{db},
		&userstore{db},
		&repostore{db},
		&keystore{db},
		&buildstore{db},
		&jobstore{db},
		&logstore{db},
	)
}

func From(db *sql.DB) store.Store {
	var driver string
	return store.New(
		driver,
		&nodestore{db},
		&userstore{db},
		&repostore{db},
		&keystore{db},
		&buildstore{db},
		&jobstore{db},
		&logstore{db},
	)
}

// Open opens a new database connection with the specified
// driver and connection string and returns a store.
func Open(driver, config string) *sql.DB {
	db, err := sql.Open(driver, config)
	if err != nil {
		log.Errorln(err)
		log.Fatalln("database connection failed")
	}
	if driver == "mysql" {
		// per issue https://github.com/go-sql-driver/mysql/issues/257
		db.SetMaxIdleConns(0)
	}

	setupMeddler(driver)

	if err := pingDatabase(db); err != nil {
		log.Errorln(err)
		log.Fatalln("database ping attempts failed")
	}

	if err := setupDatabase(driver, db); err != nil {
		log.Errorln(err)
		log.Fatalln("migration failed")
	}
	cleanupDatabase(db)
	return db
}

// OpenTest opens a new database connection for testing purposes.
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
	return Open(driver, config)
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
		log.Infof("database ping failed. retry in 1s")
		time.Sleep(time.Second)
	}
	return
}

// helper function to setup the databsae by performing
// automated database migration steps.
func setupDatabase(driver string, db *sql.DB) error {
	var migrations = &migrate.AssetMigrationSource{
		Asset:    migration.Asset,
		AssetDir: migration.AssetDir,
		Dir:      driver,
	}
	_, err := migrate.Exec(db, driver, migrations, migrate.Up)
	return err
}

// helper function to avoid stuck jobs when Drone unexpectedly
// restarts. This is a temp fix for https://github.com/drone/drone/issues/1195
func cleanupDatabase(db *sql.DB) {
	db.Exec("update builds set build_status = 'error' where build_status IN ('pending','running')")
	db.Exec("update jobs set job_status = 'error' where job_status IN ('pending','running')")
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
