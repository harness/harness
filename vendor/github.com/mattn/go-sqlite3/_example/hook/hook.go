package main

import (
	"database/sql"
	"github.com/mattn/go-sqlite3"
	"log"
	"os"
)

func main() {
	sqlite3conn := []*sqlite3.SQLiteConn{}
	sql.Register("sqlite3_with_hook_example",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				sqlite3conn = append(sqlite3conn, conn)
				return nil
			},
		})
	os.Remove("./foo.db")
	os.Remove("./bar.db")

	destDb, err := sql.Open("sqlite3_with_hook_example", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer destDb.Close()
	destDb.Ping()

	_, err = destDb.Exec("create table foo(id int, value text)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = destDb.Exec("insert into foo values(1, 'foo')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = destDb.Exec("insert into foo values(2, 'bar')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = destDb.Query("select * from foo")
	if err != nil {
		log.Fatal(err)
	}
	srcDb, err := sql.Open("sqlite3_with_hook_example", "./bar.db")
	if err != nil {
		log.Fatal(err)
	}
	defer srcDb.Close()
	srcDb.Ping()

	bk, err := sqlite3conn[1].Backup("main", sqlite3conn[0], "main")
	if err != nil {
		log.Fatal(err)
	}

	_, err = bk.Step(-1)
	if err != nil {
		log.Fatal(err)
	}
	_, err = destDb.Query("select * from foo")
	if err != nil {
		log.Fatal(err)
	}
	_, err = destDb.Exec("insert into foo values(3, 'bar')")
	if err != nil {
		log.Fatal(err)
	}

	bk.Finish()
}
