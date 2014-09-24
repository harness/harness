package migrationutil

import (
	"database/sql"
	"fmt"
	"strings"
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
	return p.Tx.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)",
		tableName, strings.Join(args, ", ")))
}

func (p *postgresqlDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return p.Tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tableName, newName))
}

func (p *postgresqlDriver) DropTable(tableName string) (sql.Result, error) {
	return p.Tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
}

func (p *postgresqlDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return p.Tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnSpec))
}

func (p *postgresqlDriver) ChangeColumn(tableName, columnName, newSpecs string) (sql.Result, error) {
	return p.Tx.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", tableName, columnName, newSpecs))
}

func (p *postgresqlDriver) DropColumns(tableName string, columnsToDrop ...string) (sql.Result, error) {
	if len(columnsToDrop) == 0 {
		return nil, fmt.Errorf("No columns to drop.")
	}
	for k, v := range columnsToDrop {
		columnsToDrop[k] = fmt.Sprintf("DROP COLUMN %s", v)
	}
	return p.Tx.Exec(fmt.Sprintf("ALTER TABLE %s %s", tableName, strings.Join(columnsToDrop, ", ")))
}

func (p *postgresqlDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	var colspec []string
	for k, v := range columnChanges {
		colspec = append(colspec, fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;", tableName, k, v))
	}

	return p.Tx.Exec(strings.Join(colspec, "\n"))
}

func (p *postgresqlDriver) AddIndex(tableName string, columns []string, flags ...string) (sql.Result, error) {
	flag := ""
	if len(flags) > 0 {
		if strings.ToUpper(flags[0]) == "UNIQUE" {
			flag = flags[0]
		}
	}

	return p.Tx.Exec(fmt.Sprintf("CREATE %s INDEX %s ON %s (%s)", flag, indexName(tableName, columns),
		tableName, strings.Join(columns, ", ")))
}

func (p *postgresqlDriver) DropIndex(tableName string, columns []string) (sql.Result, error) {
	return p.Tx.Exec(fmt.Sprintf("DROP INDEX %s", indexName(tableName, columns)))
}
