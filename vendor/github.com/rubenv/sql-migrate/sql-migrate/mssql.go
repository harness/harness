// +build go1.3

package main

import (
	_ "github.com/denisenkom/go-mssqldb"
	"gopkg.in/gorp.v1"
)

func init() {
	dialects["mssql"] = gorp.SqlServerDialect{}
}
