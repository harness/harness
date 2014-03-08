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

func (p *mysqlDriver) CreateTable(tableName string, args []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *mysqlDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *mysqlDriver) DropTable(tableName string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *mysqlDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *mysqlDriver) DropColumns(tableName string, columnsToDrop []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *mysqlDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}
