package helper

import (
	"github.com/jmoiron/sqlx"
)

var Driver string

func Rebind(query string) string {
	return sqlx.Rebind(sqlx.BindType(Driver), query)
}
