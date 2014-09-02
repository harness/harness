package migrationutil

import (
	"database/sql"
)

// Operation interface covers basic migration operations.
// Implementation details is specific for each database,
// see migrate/sqlite.go for implementation reference.
type Operation interface {

	// CreateTable may be used to create a table named `tableName`
	// with its columns specification listed in `args` as an array of string
	CreateTable(tableName string, args []string) (sql.Result, error)

	// RenameTable simply rename table from `tableName` to `newName`
	RenameTable(tableName, newName string) (sql.Result, error)

	// DropTable drops table named `tableName`
	DropTable(tableName string) (sql.Result, error)

	// AddColumn adds single new column to `tableName`, columnSpec is
	// a standard column definition (column name included) which may looks like this:
	//
	//     mg.AddColumn("example", "email VARCHAR(255) UNIQUE")
	//
	// it's equivalent to:
	//
	//     mg.AddColumn("example", mg.T.String("email", UNIQUE))
	//
	AddColumn(tableName, columnSpec string) (sql.Result, error)

	// ChangeColumn may be used to change the type of a column
	// `newType` should always specify the column's new type even
	// if the type is not meant to be change. Eg.
	//
	//     mg.ChangeColumn("example", "name", "VARCHAR(255) UNIQUE")
	//
	ChangeColumn(tableName, columnName, newType string) (sql.Result, error)

	// DropColumns drops a list of columns
	DropColumns(tableName string, columnsToDrop ...string) (sql.Result, error)

	// RenameColumns will rename columns listed in `columnChanges`
	RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error)

	// AddIndex adds index on `tableName` indexed by `columns`
	AddIndex(tableName string, columns []string, flags ...string) (sql.Result, error)

	// DropIndex drops index indexed by `columns` from `tableName`
	DropIndex(tableName string, columns []string) (sql.Result, error)
}

// MigrationDriver drives migration script by injecting transaction object (*sql.Tx),
// `Operation` implementation and column type helper.
type MigrationDriver struct {
	Operation
	T  *columnType
	Tx *sql.Tx
}

// DriverBuilder is a constructor for MigrationDriver
type DriverBuilder func(tx *sql.Tx) *MigrationDriver
