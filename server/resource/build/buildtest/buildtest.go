package committest

import (
	"database/sql"
)

func Load(db *sql.DB) {
	db.Exec("insert into builds values (1, 2, 1, '', 'Success', 'some output', 1398065345, 1398069999, 854, 1398065343, 1398065344);")
	db.Exec("insert into builds values (2, 2, 2, '', 'Success', 'some output', 1398065345, 1398069999, 854, 1398065343, 1398065344);")
	db.Exec("insert into builds values (3, 2, 3, '', 'Success', 'some output', 1398065345, 1398069999, 854, 1398065343, 1398065344);")
	db.Exec("insert into builds values (4, 1, 1, '', 'Success', 'some output', 1398065345, 1398069999, 854, 1398065343, 1398065344);")
	db.Exec("insert into builds values (5, 3, 1, '', 'Started', 'some output', 1398065345,          0,   0, 1398065343, 1398065344);")
}
