package migrationutil

import (
	"database/sql"
	"fmt"
	"strings"
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
	return m.Tx.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s) ROW_FORMAT=DYNAMIC",
		tableName, strings.Join(args, ", ")))
}

func (m *mysqlDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return m.Tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tableName, newName))
}

func (m *mysqlDriver) DropTable(tableName string) (sql.Result, error) {
	return m.Tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
}

func (m *mysqlDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return m.Tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN (%s)", tableName, columnSpec))
}

func (m *mysqlDriver) ChangeColumn(tableName, columnName, newSpecs string) (sql.Result, error) {
	return m.Tx.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY %s %s", tableName, columnName, newSpecs))
}

func (m *mysqlDriver) DropColumns(tableName string, columnsToDrop ...string) (sql.Result, error) {
	if len(columnsToDrop) == 0 {
		return nil, fmt.Errorf("No columns to drop.")
	}
	for k, v := range columnsToDrop {
		columnsToDrop[k] = fmt.Sprintf("DROP %s", v)
	}
	return m.Tx.Exec(fmt.Sprintf("ALTER TABLE %s %s", tableName, strings.Join(columnsToDrop, ", ")))
}

func (m *mysqlDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	var columns []string

	tableSQL, err := m.getTableDefinition(tableName)
	if err != nil {
		return nil, err
	}

	columns, err = fetchColumns(tableSQL)
	if err != nil {
		return nil, err
	}

	var colspec []string
	for k, v := range columnChanges {
		for _, col := range columns {
			col = strings.Trim(col, " \n")
			cols := strings.SplitN(col, " ", 2)
			if quote(k) == cols[0] {
				colspec = append(colspec, fmt.Sprintf("CHANGE %s %s %s", k, v, cols[1]))
				break
			}
		}
	}

	return m.Tx.Exec(fmt.Sprintf("ALTER TABLE %s %s", tableName, strings.Join(colspec, ", ")))
}

func (m *mysqlDriver) AddIndex(tableName string, columns []string, flags ...string) (sql.Result, error) {
	flag := ""
	if len(flags) > 0 {
		switch strings.ToUpper(flags[0]) {
		case "UNIQUE":
			fallthrough
		case "FULLTEXT":
			fallthrough
		case "SPATIAL":
			flag = flags[0]
		}
	}
	return m.Tx.Exec(fmt.Sprintf("CREATE %s INDEX %s ON %s (%s)", flag,
		indexName(tableName, columns), tableName, strings.Join(columns, ", ")))
}

func (m *mysqlDriver) DropIndex(tableName string, columns []string) (sql.Result, error) {
	return m.Tx.Exec(fmt.Sprintf("DROP INDEX %s on %s", indexName(tableName, columns), tableName))
}

func (m *mysqlDriver) getTableDefinition(tableName string) (string, error) {
	var name, def string
	st := fmt.Sprintf("SHOW CREATE TABLE %s", tableName)
	if err := m.Tx.QueryRow(st).Scan(&name, &def); err != nil {
		return "", err
	}
	return def, nil
}

func quote(name string) string {
	return fmt.Sprintf("`%s`", name)
}
