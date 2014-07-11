package testdata

import (
	"database/sql"
	"fmt"
)

var stmts = []string{
	// insert user entries
	"insert into users values (1, 'github.com', 'smellypooper',  'f0b461ca586c27872b43a0685cbc2847', '976f22a5eef7caacb7e678d6c52f49b1', 'Dr. Cooper',       'drcooper@caltech.edu', 'b9015b0857e16ac4d94a0ffd9a0b79c8', 'e42080dddf012c718e476da161d21ad5', 1, 1, 0, 1398065343, 1398065344, 1398065345);",
	"insert into users values (2, 'github.com', 'lhofstadter',   'e4105c3059ac4c466594932dc9a4ffb2', '2257216903d9cd0d3d24772132febf52', 'Dr. Hofstadter',   'leanard@caltech.edu',  '23dde632fdece6880f4ff03bb20f05d7', 'a5ad0d75f317f0b0a5dfdb68e5a3079e', 1, 1, 0, 1398065343, 1398065344, 1398065345);",
	"insert into users values (3, 'gitlab.com', 'browndynamite', '4821477cc26a0c8c80c6c9b568d98e32', '1dd52c37cf5c63fe5abfd047b5b74a31', 'Dr. Koothrappali', 'rajesh@caltech.edu',   'f9133051f480b7ea88848b9f0a079dae', '7a50ede04637d4a8fce532c7d511226b', 1, 1, 0, 1398065343, 1398065344, 1398065345);",
	"insert into users values (4, 'github.com', 'mrwolowitz',    '1f6a80bde960e6913bf9b7e61eadd068', '74c40472494ba7f9f6c3ae061ff799ed', 'Mr. Wolowitz',     'wolowitz@caltech.edu', 'ea250570c794d84dc583421bb717be82', '3bd7e7d7411b2978e45919c9ad419984', 1, 1, 0, 1398065343, 1398065344, 1398065345);",

	// insert repository entries
	"insert into repos values (1, 0, 'github.com', 'github.com', 'lhofstadter',   'lenwoloppali', '', 'git://github.com/lhofstadter/lenwoloppali.git', '', '', 1, 1, 1, 1, 1, 'publickey', 'privatekey', 'params', 900, 1398065343, 1398065344);",
	"insert into repos values (2, 0, 'github.com', 'github.com', 'browndynamite', 'lenwoloppali', '', 'git://github.com/browndynamite/lenwoloppali.git', '', '', 1, 1, 1, 1, 1, 'publickey', 'privatekey', 'params', 900, 1398065343, 1398065344);",
	"insert into repos values (3, 0, 'gitlab.com', 'gitlab.com', 'browndynamite', 'lenwoloppali', '', 'git://gitlab.com/browndynamite/lenwoloppali.git', '', '', 1, 1, 1, 1, 1, 'publickey', 'privatekey', 'params', 900, 1398065343, 1398065344);",

	// insert user + repository permission entries
	"insert into perms values (1, 101, 200, 1, 1, 1, 1398065343, 1398065344);",
	"insert into perms values (2, 102, 200, 1, 1, 0, 1398065343, 1398065344);",
	"insert into perms values (3, 103, 200, 1, 0, 0, 1398065343, 1398065344);",
	"insert into perms values (4, 1, 1, 1, 1, 1, 1398065343, 1398065344);",
	"insert into perms values (5, 1, 2, 1, 1, 0, 1398065343, 1398065344);",

	// insert commit entries
	"insert into commits values (1, 2, 'Success', 1398065345, 1398069999, 854, '4e81eca185897c2d0cf81f5bc12623550c2ef952', 'dev',    '3', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 01:00:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);",
	"insert into commits values (2, 2, 'Success', 1398065345, 1398069999, 854, '4e81eca185897c2d0cf81f5bc12623550c2ef952', 'master', '4', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 01:01:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);",
	"insert into commits values (3, 2, 'Success', 1398065345, 1398069999, 854, '7253f6545caed41fb8f5a6fcdb3abc0b81fa9dbf', 'master', '5', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 01:02:38 2014 -0700', 'a commit message', '', 1398065343, 1398065344);",
	"insert into commits values (4, 1, 'Success', 1398065345, 1398069999, 854, 'd12c9e5a11982f71796ad909c93551b16fba053e', 'dev',     '', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 02:00:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);",
	"insert into commits values (5, 1, 'Started', 1398065345,          0,   0, '85f8c029b902ed9400bc600bac301a0aadb144ac', 'master',  '', 'drcooper@caltech.edu', 'ab23a88a3ed77ecdfeb894c0eaf2817a', 'Wed Apr 23 03:00:00 2014 -0700', 'a commit message', '', 1398065343, 1398065344);",

	// insert commit console output
	"insert into output values (1, 1, 'sample console output');",
	"insert into output values (2, 2, 'sample console output.....');",

	// insert server entries
	"insert into servers values (1, 'docker1', 'tcp://127.0.0.1:4243', 'root', 'pa55word', '/path/to/cert.key');",
	"insert into servers values (2, 'docker2', 'tcp://127.0.0.1:4243', 'root', 'pa55word', '/path/to/cert.key');",

	// insert remote entries
	"insert into remotes values (1, 'enterprise.github.com', 'github.drone.io', 'https://github.drone.io', 'https://github.drone.io/v3/api', 'f0b461ca586c27872b43a0685cbc2847', '976f22a5eef7caacb7e678d6c52f49b1', '1');",
	"insert into remotes values (2, 'github.com',            'github.com',      'https://github.io',       'https://api.github.com',          'a0b461ca586c27872b43a0685cbc2847', 'a76f22a5eef7caacb7e678d6c52f49b1', '0');",
}

// Load will populate the database with a fixed dataset for
// unit testing purposes.
func Load(db *sql.DB) {
	// loop through insert statements and execute
	for _, stmt := range stmts {
		_, err := db.Exec(stmt)
		if err != nil {
			fmt.Printf("Error executing Query %s\n %s\n", stmt, err)
		}
	}
}
