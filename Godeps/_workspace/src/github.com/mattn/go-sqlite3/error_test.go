package sqlite3

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestSimpleError(t *testing.T) {
	e := ErrError.Error()
	if e != "SQL logic error or missing database" {
		t.Error("wrong error code:" + e)
	}
}

func TestCorruptDbErrors(t *testing.T) {
	dirName, err := ioutil.TempDir("", "sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirName)

	dbFileName := path.Join(dirName, "test.db")
	f, err := os.Create(dbFileName)
	if err != nil {
		t.Error(err)
	}
	f.Write([]byte{1, 2, 3, 4, 5})
	f.Close()

	db, err := sql.Open("sqlite3", dbFileName)
	if err == nil {
		_, err = db.Exec("drop table foo")
	}

	sqliteErr := err.(Error)
	if sqliteErr.Code != ErrNotADB {
		t.Error("wrong error code for corrupted DB")
	}
	if err.Error() == "" {
		t.Error("wrong error string for corrupted DB")
	}
	db.Close()
}

func TestSqlLogicErrors(t *testing.T) {
	dirName, err := ioutil.TempDir("", "sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirName)

	dbFileName := path.Join(dirName, "test.db")
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		t.Error(err)
	}

	_, err = db.Exec("CREATE TABLE Foo (id INT PRIMARY KEY)")
	if err != nil {
		t.Error(err)
	}

	const expectedErr = "table Foo already exists"
	_, err = db.Exec("CREATE TABLE Foo (id INT PRIMARY KEY)")
	if err.Error() != expectedErr {
		t.Errorf("Unexpected error: %s, expected %s", err.Error(), expectedErr)
	}
}
