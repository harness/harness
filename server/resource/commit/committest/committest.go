package committest

import (
	"database/sql"
)

func Load(db *sql.DB) {
	db.Exec("insert into commits values (1, 2, 'Success', 1398065345, 1398069999, 854, '4e81eca185897c2d0cf81f5bc12623550c2ef952', 'dev',    '3', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 01:00:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);")
	db.Exec("insert into commits values (2, 2, 'Success', 1398065345, 1398069999, 854, '4e81eca185897c2d0cf81f5bc12623550c2ef952', 'master', '4', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 01:01:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);")
	db.Exec("insert into commits values (3, 2, 'Success', 1398065345, 1398069999, 854, '7253f6545caed41fb8f5a6fcdb3abc0b81fa9dbf', 'master', '5', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 01:02:38 2014 -0700', 'a commit message', '', 1398065343, 1398065344);")
	db.Exec("insert into commits values (4, 1, 'Success', 1398065345, 1398069999, 854, 'd12c9e5a11982f71796ad909c93551b16fba053e', 'dev',     '', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 02:00:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);")
	db.Exec("insert into commits values (5, 1, 'Started', 1398065345,          0,   0, '85f8c029b902ed9400bc600bac301a0aadb144ac', 'master',  '', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 03:00:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);")
}
