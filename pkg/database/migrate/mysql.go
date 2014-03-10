package migrate

import (
	"database/sql"
	"errors"
)

type mysqlDriver struct {
	Tx *sql.Tx
}

func MySQL(tx *sql.Tx) *MigrationDriver {
	return &MigrationDriver{
		Tx:        tx,
		Operation: &mysqlDriver{Tx: tx},
	}
}

func (m *mysqlDriver) CreateTable(tableName string, args []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) DropTable(tableName string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) ChangeColumn(tableName, columnName, newSpecs string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) DropColumns(tableName string, columnsToDrop []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) AddIndex(tableName string, columns []string, flag string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (m *mysqlDriver) DropIndex(tableName string, columns []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}
