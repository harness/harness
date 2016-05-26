// +build !cgo

package datastore

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)
