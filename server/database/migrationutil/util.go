package migrationutil

import (
	"fmt"
	"strings"

	"github.com/dchest/uniuri"
)

func fetchColumns(sql string) ([]string, error) {
	if !strings.HasPrefix(sql, "CREATE ") {
		return []string{}, fmt.Errorf("Sql input is not a DDL statement.")
	}

	parenIdx := strings.Index(sql, "(")
	return strings.Split(sql[parenIdx+1:strings.LastIndex(sql, ")")], ","), nil
}

func selectName(columns []string) []string {
	var results []string
	for _, column := range columns {
		col := strings.SplitN(strings.Trim(column, " \n\t"), " ", 2)
		results = append(results, col[0])
	}
	return results
}

func setForUpdate(left []string, right []string) string {
	var results []string
	for k, str := range left {
		results = append(results, fmt.Sprintf("%s = %s", str, right[k]))
	}
	return strings.Join(results, ", ")
}

func proxyName(tableName string) string {
	return fmt.Sprintf("%s_%s", tableName, uniuri.NewLen(16))
}

func indexName(tableName string, columns []string) string {
	return fmt.Sprintf("idx_%s_on_%s", tableName, strings.Join(columns, "_and_"))
}
