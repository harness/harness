package main

import (
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"log"
)

func main() {
	sql.Register("sqlite3_with_extensions",
		&sqlite3.SQLiteDriver{
			Extensions: []string{
				"sqlite3_mod_vtable",
			},
		})

	db, err := sql.Open("sqlite3_with_extensions", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec("create virtual table repo using github(id, full_name, description, html_url)")

	rows, err := db.Query("select id, full_name, description, html_url from repo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, full_name, description, html_url string
		rows.Scan(&id, &full_name, &description, &html_url)
		fmt.Printf("%s: %s\n\t%s\n\t%s\n\n", id, full_name, description, html_url)
	}
}
