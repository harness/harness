package permdata

import (
	"database/sql"
)

func Load(db *sql.DB) {
	db.Exec("insert into perms values (1, 101, 200, 1, 1, 1, 1398065343, 1398065344);")
	db.Exec("insert into perms values (2, 102, 200, 1, 1, 0, 1398065343, 1398065344);")
	db.Exec("insert into perms values (3, 103, 200, 1, 0, 0, 1398065343, 1398065344);")
}
