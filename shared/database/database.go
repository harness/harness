package database

//go:generate go-bindata -pkg database -o database_gen.go sqlite3/ mysql/ postgres/

import (
	"database/sql"

	"github.com/CiscoCloud/drone/shared/envconfig"

	log "github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rubenv/sql-migrate"
)

func Load(env envconfig.Env) *sql.DB {
	var (
		driver = env.String("DATABASE_DRIVER", "sqlite3")
		config = env.String("DATABASE_CONFIG", "drone.sqlite")
	)

	log.Infof("using database driver %s", driver)
	log.Infof("using database config %s", config)

	return Open(driver, config)
}

// Open opens a database connection, runs the database migrations, and returns
// the database connection. Any errors connecting to the database or executing
// migrations will cause the application to exit.
func Open(driver, config string) *sql.DB {
	var db, err = sql.Open(driver, config)
	if err != nil {
		log.Errorln(err)
		log.Fatalln("database connection failed")
	}

	var migrations = &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      driver,
	}

	_, err = migrate.Exec(db, driver, migrations, migrate.Up)
	if err != nil {
		log.Errorln(err)
		log.Fatalln("migration failed")
	}
	return db
}
