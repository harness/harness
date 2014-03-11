package database

import (
	"database/sql"
	"fmt"

	"github.com/drone/drone/pkg/database/migrate"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/russross/meddler"
)

// global instance of our database connection.
var db *sql.DB

// Init connects to database and performs migration if necessary.
//
// Database driver name and data source information is provided by user
// from within command line, and error checking is deferred to sql.Open.
//
// Init will just bail out and returns error if driver name
// is not listed, no fallback nor default driver sets here.
func Init(name, datasource string) error {
	var err error
	driver := map[string]struct {
		Md *meddler.Database
		Mg migrate.DriverBuilder
	}{
		"sqlite3": {
			meddler.SQLite,
			migrate.SQLite,
		},
		"mysql": {
			meddler.MySQL,
			migrate.MySQL,
		},
	}

	if drv, ok := driver[name]; ok {
		meddler.Default = drv.Md
		migrate.Driver = drv.Mg
	} else {
		return fmt.Errorf("%s driver not found", name)
	}

	db, err = sql.Open(name, datasource)
	if err != nil {
		return err
	}

	migration := migrate.New(db)
	migration.All().Migrate()
	return nil
}

func Close() {
	db.Close()
}
