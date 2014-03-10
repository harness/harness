package migrate

import (
	"database/sql"
)

// Operation interface covers basic migration operations.
// Implementation details is specific for each database,
// see migrate/sqlite.go for implementation reference.
type Operation interface {
	CreateTable(tableName string, args []string) (sql.Result, error)

	RenameTable(tableName, newName string) (sql.Result, error)

	DropTable(tableName string) (sql.Result, error)

	AddColumn(tableName, columnSpec string) (sql.Result, error)

	ChangeColumn(tableName, columnName, newType string) (sql.Result, error)

	DropColumns(tableName string, columnsToDrop []string) (sql.Result, error)

	RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error)

	AddIndex(tableName string, columns []string, flag string) (sql.Result, error)

	DropIndex(tableName string, columns []string) (sql.Result, error)
}

type MigrationDriver struct {
	Tx *sql.Tx
	Operation
}

type DriverBuilder func(tx *sql.Tx) *MigrationDriver
