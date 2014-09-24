package migrationutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/drone/drone/server/helper"
)

const (
	NULL int = iota
	NOTNULL
)

// columnType will be injected to migration script
// along with MigrationDriver. `AttrMap` is used to
// defines distinct column's attribute between database
// implementation. e.g. 'AUTOINCREMENT' in sqlite and
// 'AUTO_INCREMENT' in mysql.
type columnType struct {
	AttrMap map[int]string
}

// defaultMap defines default values for column's attribute
// lookup.
var defaultMap = map[int]string{
	NULL:    "NULL",
	NOTNULL: "NOT NULL",
}

func (c *columnType) Pk(colName string) string {
	switch helper.Driver {
	case "sqlite3":
		return fmt.Sprintf("%s INTEGER PRIMARY KEY AUTOINCREMENT", colName)
	default:
		return fmt.Sprintf("%s SERIAL PRIMARY KEY", colName)
	}
}

// Integer returns column definition for INTEGER typed column.
// Additional attributes may be specified as string or predefined key
// listed in defaultMap.
func (c *columnType) Integer(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s INTEGER %s", colName, c.parseAttr(spec))
}

// String returns column definition for VARCHAR(255) typed column.
func (c *columnType) String(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s VARCHAR(255) %s", colName, c.parseAttr(spec))
}

// Text returns column definition for TEXT typed column.
func (c *columnType) Text(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s TEXT %s", colName, c.parseAttr(spec))
}

// Blob returns column definition for BLOB typed column
func (c *columnType) Blob(colName string, spec ...interface{}) string {
	switch helper.Driver {
	case "postgres":
		return fmt.Sprintf("%s BYTEA %s", colName, c.parseAttr(spec))
	default:
		return fmt.Sprintf("%s BLOB %s", colName, c.parseAttr(spec))
	}
}

// Timestamp returns column definition for TIMESTAMP typed column
func (c *columnType) Timestamp(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s TIMESTAMP %s", colName, c.parseAttr(spec))
}

// Bool returns column definition for BOOLEAN typed column
func (c *columnType) Bool(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s BOOLEAN %s", colName, c.parseAttr(spec))
}

// Varchar returns column definition for VARCHAR typed column.
// column's max length is specified as `length`.
func (c *columnType) Varchar(colName string, length int, spec ...interface{}) string {
	return fmt.Sprintf("%s VARCHAR(%d) %s", colName, length, c.parseAttr(spec))
}

// attr returns string representation of column attribute specified as key for defaultMap.
func (c *columnType) attr(flag int) string {
	if v, ok := c.AttrMap[flag]; ok {
		return v
	}
	return defaultMap[flag]
}

// parseAttr reflects spec value for its type and returns the string
// representation returned by `attr`
func (c *columnType) parseAttr(spec []interface{}) string {
	var attrs []string
	for _, v := range spec {
		switch reflect.ValueOf(v).Kind() {
		case reflect.Int:
			attrs = append(attrs, c.attr(v.(int)))
		case reflect.String:
			attrs = append(attrs, v.(string))
		}
	}
	return strings.Join(attrs, " ")
}
