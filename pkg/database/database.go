package database

import (
	"database/sql"
	"log"

	"github.com/drone/drone/pkg/database/schema"
)

// global instance of our database connection.
var db *sql.DB

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
