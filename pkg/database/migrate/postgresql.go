package migrate

import (
	"database/sql"
	"errors"
)

type postgresqlDriver struct {
	Tx *sql.Tx
}

func PostgreSQL(tx *sql.Tx) *MigrationDriver {
	return &MigrationDriver{
		Tx:        tx,
		Operation: &postgresqlDriver{Tx: tx},
	}
}

func (p *postgresqlDriver) CreateTable(tableName string, args []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) DropTable(tableName string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) ChangeColumn(tableName, columnName, newSpecs string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) DropColumns(tableName string, columnsToDrop ...string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) AddIndex(tableName string, columns []string, flags ...string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}

func (p *postgresqlDriver) DropIndex(tableName string, columns []string) (sql.Result, error) {
	return nil, errors.New("not implemented yet")
}
