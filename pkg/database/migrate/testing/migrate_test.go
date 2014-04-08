package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	. "github.com/drone/drone/pkg/database/migrate"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"
)

var (
	db          *sql.DB
	driver, dsn string

	dbname = "drone_test"
)

var sqliteTestSchema = `
CREATE TABLE samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	imel VARCHAR(255) UNIQUE,
	name VARCHAR(255)
);
`

var mysqlTestSchema = `
CREATE TABLE samples (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	imel VARCHAR(255) UNIQUE,
	name VARCHAR(255)
)
`

var dataDump = []string{
	`INSERT INTO samples (imel, name) VALUES ('test@example.com', 'Test Tester');`,
	`INSERT INTO samples (imel, name) VALUES ('foo@bar.com', 'Foo Bar');`,
	`INSERT INTO samples (imel, name) VALUES ('crash@bandicoot.io', 'Crash Bandicoot');`,
}

func TestMigrateCreateTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	mgr := New(db)
	if err := mgr.Add(&revision1{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	sample := Sample{
		ID:   1,
		Imel: "test@example.com",
		Name: "Test Tester",
	}
	if err := meddler.Save(db, "samples", &sample); err != nil {
		t.Fatalf("Can not save data: %q", err)
	}
}

func TestMigrateExistingCreateTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	var testSchema string
	if driver == "mysql" {
		testSchema = mysqlTestSchema
	} else {
		testSchema = sqliteTestSchema
	}

	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("Can not create database: %q", err)
	}

	mgr := New(db)
	rev := &revision1{}
	if err := mgr.Add(rev).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	var current int64
	db.QueryRow("SELECT max(revision) FROM migration").Scan(&current)
	if current != rev.Revision() {
		t.Fatalf("Did not successfully migrate")
	}
}

func TestMigrateRenameTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	mgr := New(db)
	if err := mgr.Add(&revision1{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	loadFixture(t)

	if err := mgr.Add(&revision2{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	sample := Sample{}
	if err := meddler.QueryRow(db, &sample, `SELECT * FROM examples WHERE id = ?`, 2); err != nil {
		t.Fatalf("Can not fetch data: %q", err)
	}

	if sample.Imel != "foo@bar.com" {
		t.Errorf("Column doesn't match. Expect: %s, got: %s", "foo@bar.com", sample.Imel)
	}
}

type TableInfo struct {
	CID       int64       `meddler:"cid,pk"`
	Name      string      `meddler:"name"`
	Type      string      `meddler:"type"`
	Notnull   bool        `meddler:"notnull"`
	DfltValue interface{} `meddler:"dflt_value"`
	PK        bool        `meddler:"pk"`
}

type MysqlTableInfo struct {
	Field   string      `meddler:"Field"`
	Type    string      `meddler:"Type"`
	Null    string      `meddler:"Null"`
	Key     interface{} `meddler:"Key"`
	Default interface{} `meddler:"Default"`
	Extra   interface{} `meddler:"Extra"`
}

func TestMigrateAddRemoveColumns(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	mgr := New(db)
	if err := mgr.Add(&revision1{}, &revision3{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	switch driver {
	case "mysql":
		var columns []*MysqlTableInfo
		if err := meddler.QueryAll(db, &columns, `SHOW COLUMNS FROM samples`); err != nil {
			t.Fatalf("Can not access table infor: %q", err)
		}

		if len(columns) < 5 {
			t.Errorf("Expect length columns: %d\nGot: %d", 5, len(columns))
		}
	default:
		var columns []*TableInfo
		if err := meddler.QueryAll(db, &columns, `PRAGMA table_info(samples);`); err != nil {
			t.Fatalf("Can not access table info: %q", err)
		}

		if len(columns) < 5 {
			t.Errorf("Expect length columns: %d\nGot: %d", 5, len(columns))
		}
	}

	var row = AddColumnSample{
		ID:   33,
		Name: "Foo",
		Imel: "foo@bar.com",
		Url:  "http://example.com",
		Num:  42,
	}
	if err := meddler.Save(db, "samples", &row); err != nil {
		t.Fatalf("Can not save into database: %q", err)
	}

	if err := mgr.MigrateTo(1); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	switch driver {
	case "mysql":
		var columns []*MysqlTableInfo
		if err := meddler.QueryAll(db, &columns, `SHOW COLUMNS FROM samples`); err != nil {
			t.Fatalf("Can not access table infor: %q", err)
		}

		if len(columns) != 3 {
			t.Errorf("Expect length columns: %d\nGot: %d", 3, len(columns))
		}
	default:
		var columns []*TableInfo
		if err := meddler.QueryAll(db, &columns, `PRAGMA table_info(samples);`); err != nil {
			t.Fatalf("Can not access table info: %q", err)
		}

		if len(columns) != 3 {
			t.Errorf("Expect length columns: %d\nGot: %d", 3, len(columns))
		}
	}

}

func TestRenameColumn(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	mgr := New(db)
	if err := mgr.Add(&revision1{}, &revision4{}).MigrateTo(1); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	loadFixture(t)

	if err := mgr.MigrateTo(4); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	row := RenameSample{}
	if err := meddler.QueryRow(db, &row, `SELECT * FROM samples WHERE id = 3;`); err != nil {
		t.Fatalf("Can not query database: %q", err)
	}

	if row.Email != "crash@bandicoot.io" {
		t.Errorf("Expect %s, got %s", "crash@bandicoot.io", row.Email)
	}
}

func TestMigrateExistingTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	var testSchema string
	if driver == "mysql" {
		testSchema = mysqlTestSchema
	} else {
		testSchema = sqliteTestSchema
	}

	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("Can not create database: %q", err)
	}

	loadFixture(t)

	mgr := New(db)
	if err := mgr.Add(&revision4{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	var rows []*RenameSample
	if err := meddler.QueryAll(db, &rows, `SELECT * from samples;`); err != nil {
		t.Fatalf("Can not query database: %q", err)
	}

	if len(rows) != 3 {
		t.Errorf("Expect rows length = %d, got %d", 3, len(rows))
	}

	if rows[1].Email != "foo@bar.com" {
		t.Errorf("Expect email = %s, got %s", "foo@bar.com", rows[1].Email)
	}
}

type sqliteMaster struct {
	Sql interface{} `meddler:"sql"`
}

func TestIndexOperations(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	mgr := New(db)

	// Migrate, create index
	if err := mgr.Add(&revision1{}, &revision3{}, &revision5{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	var esquel []*sqliteMaster
	var mysquel struct {
		Table       string `meddler:"Table"`
		CreateTable string `meddler:"Create Table"`
	}
	switch driver {
	case "mysql":
		query := `SHOW CREATE TABLE samples`
		if err := meddler.QueryRow(db, &mysquel, query); err != nil {
			t.Fatalf("Can not fetch table definition: %q", err)
		}

		if !strings.Contains(mysquel.CreateTable, "KEY `idx_samples_on_url_and_name` (`url`,`name`)") {
			t.Errorf("Can not find index, got: %q", mysquel.CreateTable)
		}

		if err := mgr.Add(&revision6{}).Migrate(); err != nil {
			t.Fatalf("Can not migrate: %q", err)
		}

		if err := meddler.QueryRow(db, &mysquel, query); err != nil {
			t.Fatalf("Can not find index: %q", err)
		}

		if !strings.Contains(mysquel.CreateTable, "KEY `idx_samples_on_url_and_name` (`host`,`name`)") {
			t.Errorf("Can not find index, got: %q", mysquel.CreateTable)
		}

		if err := mgr.Add(&revision7{}).Migrate(); err != nil {
			t.Fatalf("Can not migrate: %q", err)
		}

		if err := meddler.QueryRow(db, &mysquel, query); err != nil {
			t.Fatalf("Can not find index: %q", err)
		}

		if strings.Contains(mysquel.CreateTable, "KEY `idx_samples_on_url_and_name` (`host`,`name`)") {
			t.Errorf("Expect index to be deleted.")
		}

	default:
		// Query sqlite_master, check if index is exists.
		query := `SELECT sql FROM sqlite_master WHERE type='index' and tbl_name='samples'`
		if err := meddler.QueryAll(db, &esquel, query); err != nil {
			t.Fatalf("Can not find index: %q", err)
		}

		indexStatement := `CREATE INDEX idx_samples_on_url_and_name ON samples (url, name)`
		if string(esquel[1].Sql.([]byte)) != indexStatement {
			t.Errorf("Can not find index, got: %q", esquel[1])
		}

		// Migrate, rename indexed columns
		if err := mgr.Add(&revision6{}).Migrate(); err != nil {
			t.Fatalf("Can not migrate: %q", err)
		}

		var esquel1 []*sqliteMaster
		if err := meddler.QueryAll(db, &esquel1, query); err != nil {
			t.Fatalf("Can not find index: %q", err)
		}

		indexStatement = `CREATE INDEX idx_samples_on_host_and_name ON samples (host, name)`
		if string(esquel1[1].Sql.([]byte)) != indexStatement {
			t.Errorf("Can not find index, got: %q", esquel1[1])
		}

		if err := mgr.Add(&revision7{}).Migrate(); err != nil {
			t.Fatalf("Can not migrate: %q", err)
		}

		var esquel2 []*sqliteMaster
		if err := meddler.QueryAll(db, &esquel2, query); err != nil {
			t.Fatalf("Can not find index: %q", err)
		}

		if len(esquel2) != 1 {
			t.Errorf("Expect row length equal to %d, got %d", 1, len(esquel2))
		}
	}
}

func TestColumnRedundancy(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	migr := New(db)
	if err := migr.Add(&revision1{}, &revision8{}, &revision9{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	var dummy, query, tableSql string
	switch driver {
	case "mysql":
		query = `SHOW CREATE TABLE samples`
		if err := db.QueryRow(query).Scan(&dummy, &tableSql); err != nil {
			t.Fatalf("Can not query table's definition: %q", err)
		}
		if !strings.Contains(tableSql, "`repository`") {
			t.Errorf("Expect column with name repository")
		}
	default:
		query = `SELECT sql FROM sqlite_master where type='table' and name='samples'`
		if err := db.QueryRow(query).Scan(&tableSql); err != nil {
			t.Fatalf("Can not query sqlite_master: %q", err)
		}
		if !strings.Contains(tableSql, "repository ") {
			t.Errorf("Expect column with name repository")
		}
	}
}

func TestChangeColumnType(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	migr := New(db)
	if err := migr.Add(&revision1{}, &revision4{}, &revision10{}).Migrate(); err != nil {
		t.Fatalf("Can not migrate: %q", err)
	}

	var dummy, tableSql, query string
	switch driver {
	case "mysql":
		query = `SHOW CREATE TABLE samples`
		if err := db.QueryRow(query).Scan(&dummy, &tableSql); err != nil {
			t.Fatalf("Can not query table's definition: %q", err)
		}
		if !strings.Contains(tableSql, "`email` varchar(512)") {
			t.Errorf("Expect email type to changed: %q", tableSql)
		}
	default:
		query = `SELECT sql FROM sqlite_master where type='table' and name='samples'`
		if err := db.QueryRow(query).Scan(&tableSql); err != nil {
			t.Fatalf("Can not query sqlite_master: %q", err)
		}
		if !strings.Contains(tableSql, "email varchar(512) UNIQUE") {
			t.Errorf("Expect email type to changed: %q", tableSql)
		}
	}
}

func init() {
	if driver = os.Getenv("DB_ENV"); len(driver) == 0 {
		driver = "sqlite3"
	}
	if dsn = os.Getenv("MYSQL_LOGIN"); len(dsn) == 0 {
		dsn = ":memory:"
	} else {
		dsn = fmt.Sprintf("%s@/?parseTime=true", dsn)
	}
}

func setUp() error {
	var err error
	Driver = SQLite
	if db, err = sql.Open(driver, dsn); err != nil {
		log.Fatalf("Can't connect to database: %q", err)
	}
	if driver == "mysql" {
		Driver = MySQL
		if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname)); err != nil {
			log.Fatalf("Can't create database: %q", err)
		}
		if _, err := db.Exec(fmt.Sprintf("USE %s", dbname)); err != nil {
			log.Fatalf("Can't use database: %q", dbname)
		}
	}
	return err
}

func tearDown() {
	if driver == "mysql" {
		db.Exec(fmt.Sprintf("DROP DATABASE %s", dbname))
	}
	db.Close()
}

func loadFixture(t *testing.T) {
	for _, sql := range dataDump {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("Can not insert into database: %q", err)
		}
	}
}
