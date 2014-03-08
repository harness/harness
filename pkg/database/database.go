package database

import (
	"database/sql"
	"log"

	"github.com/drone/drone/pkg/database/migrate"
	"github.com/drone/drone/pkg/database/schema"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/russross/meddler"
)

// global instance of our database connection.
var db *sql.DB

func Init(name, datasource string) error {
	driver := map[string]struct {
		Md *meddler.Database
		Mg migrate.DriverFunction
	}{
		"sqlite3": {
			meddler.SQLite,
			migrate.SQLite,
		},
		"mysql": {
			meddler.MySQL,
			migrate.MySQL,
		},
		"postgresql": {
			meddler.PostgreSQL,
			migrate.PostgreSQL,
		},
	}

	meddler.Default = driver[name].Md
	migrate.Driver = driver[name].Mg

	db, err := sql.Open(name, datasource)
	if err != nil {
		return err
	}

	Set(db)

	migration := migrate.New(db)
	migration.All().Migrate()
	return nil
}

// Set sets the default database.
func Set(database *sql.DB) {
	// set the global database
	db = database

	// load the database schema. If this is
	// a new database all the tables and
	// indexes will be created.
	if err := schema.Load(db); err != nil {
		log.Fatal(err)
	}
}
