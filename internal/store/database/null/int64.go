// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package null

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

// Int64 represents an int64 that may be null.
type Int64 struct {
	sql.NullInt64
}

func NewInt64(i int64) Int64 {
	return Int64{
		sql.NullInt64{
			Int64: i,
			Valid: true,
		},
	}
}

// FromInt64 returns a null Int64 if the parameter is zero, a valid Int64 otherwise.
func FromInt64(i int64) Int64 {
	if i == 0 {
		return Int64{}
	}
	return NewInt64(i)
}

// FromPtrInt64 returns a null Int64 if the parameter is nil, a valid Int64 otherwise.
func FromPtrInt64(i *int64) Int64 {
	if i == nil {
		return Int64{}
	}
	return NewInt64(*i)
}

// ToInt64 converts the null.Int64 to an int64.
func (i *Int64) ToInt64() int64 {
	if !i.Valid {
		return 0
	}
	return i.Int64
}

// ToPtrInt64 converts the null.Int64 to an *int64.
func (i *Int64) ToPtrInt64() *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

// UnmarshalJSON implements json.Unmarshaler.
func (i *Int64) UnmarshalJSON(input []byte) error {
	var value interface{}
	if err := json.Unmarshal(input, &value); err != nil {
		return err
	}

	switch z := value.(type) {
	case float64: // in JSON a number is float64
		i.Int64 = int64(z)
		i.Valid = true
	default:
		i.Int64 = 0
		i.Valid = false
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (i *Int64) MarshalJSON() ([]byte, error) {
	if !i.Valid {
		return []byte(null), nil
	}
	return json.Marshal(i.Int64)
}

// Scan implements sql.Scanner interface
func (i *Int64) Scan(input interface{}) error {
	switch val := input.(type) {
	case nil:
		i.Int64, i.Valid = 0, false
	case int64:
		i.Int64, i.Valid = val, true
	case float64:
		i.Int64, i.Valid = int64(val), true
	case []byte:
		ii, err := strconv.ParseInt(string(val), 10, 64)
		i.Int64, i.Valid = ii, err == nil
	case string:
		ii, err := strconv.ParseInt(val, 10, 64)
		i.Int64, i.Valid = ii, err == nil
	default:
		return fmt.Errorf("failed to convert %v (%T) to null.Int64", input, input)
	}

	return nil
}

// Value implements driver.Valuer interface
func (i *Int64) Value() (driver.Value, error) {
	if !i.Valid {
		return nil, nil
	}
	return i.Int64, nil
}
