// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"encoding/json"

	sqlx "github.com/jmoiron/sqlx/types"
)

// EncodeToJSON accepts a generic parameter and returns
// a sqlx.JSONText object which is used to store arbitrary
// data in the DB. We absorb the error here as the value
// gets absorbed in sqlx.JSONText in case of UnsupportedValueError
// or UnsupportedTypeError.
func EncodeToJSON(v any) sqlx.JSONText {
	raw, _ := json.Marshal(v)
	return sqlx.JSONText(raw)
}
