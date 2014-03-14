package migrate

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	UNIQUE int = iota
	PRIMARYKEY
	AUTOINCREMENT
	NULL
	NOTNULL

	TSTRING
	TTEXT
)

type columnType struct {
	Driver  string
	AttrMap map[int]string
}

var defaultMap = map[int]string{
	UNIQUE:        "UNIQUE",
	PRIMARYKEY:    "PRIMARY KEY",
	AUTOINCREMENT: "AUTOINCREMENT",
	NULL:          "NULL",
	NOTNULL:       "NOT NULL",
}

func (c *columnType) Integer(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s INTEGER %s", colName, c.parseAttr(spec))
}

func (c *columnType) String(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s VARCHAR(255) %s", colName, c.parseAttr(spec))
}

func (c *columnType) Text(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s TEXT %s", colName, c.parseAttr(spec))
}

func (c *columnType) Blob(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s BLOB %s", colName, c.parseAttr(spec))
}

func (c *columnType) Timestamp(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s TIMESTAMP %s", colName, c.parseAttr(spec))
}

func (c *columnType) Bool(colName string, spec ...interface{}) string {
	return fmt.Sprintf("%s BOOLEAN %s", colName, c.parseAttr(spec))
}

func (c *columnType) Varchar(colName string, length int, spec ...interface{}) string {
	return fmt.Sprintf("%s VARCHAR(%d) %s", colName, length, c.parseAttr(spec))
}

func (c *columnType) attr(flag int) string {
	if v, ok := c.AttrMap[flag]; ok {
		return v
	}
	return defaultMap[flag]
}

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
