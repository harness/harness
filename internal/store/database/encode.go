package database

import (
	"encoding/json"

	sqlx "github.com/jmoiron/sqlx/types"
)

// EncodeToJSON accepts a generic parameter and returns
// a sqlx.JSONText object which is used to store arbitrary
// data in the DB.
func EncodeToJSON(v any) sqlx.JSONText {
	raw, _ := json.Marshal(v)
	return sqlx.JSONText(raw)
}
