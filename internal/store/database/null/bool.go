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

// Bool represents a bool that may be null.
type Bool struct {
	sql.NullBool
}

func NewBool(b bool) Bool {
	return Bool{
		sql.NullBool{
			Bool:  b,
			Valid: true,
		},
	}
}

// FromBool returns a null Bool if the parameter is false or a true Bool.
func FromBool(b bool) Bool {
	if !b {
		return Bool{}
	}
	return NewBool(b)
}

// FromPtrBool returns a null Bool if the parameter is nil, a valid Bool otherwise.
func FromPtrBool(b bool) Bool {
	if !b {
		return Bool{}
	}
	return NewBool(b)
}

// ToBool converts null.Bool to a bool.
func (b *Bool) ToBool() bool {
	if !b.Valid {
		return false
	}
	return b.Bool
}

// ToPtrBool converts null.Bool to a *bool.
func (b *Bool) ToPtrBool() *bool {
	if !b.Valid {
		return nil
	}
	return &b.Bool
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Bool) UnmarshalJSON(input []byte) error {
	var i interface{}
	if err := json.Unmarshal(input, &i); err != nil {
		return err
	}

	switch val := i.(type) {
	case bool:
		b.Bool = val
		b.Valid = true
	default:
		b.Bool = false
		b.Valid = false
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (b *Bool) MarshalJSON() ([]byte, error) {
	if !b.Valid {
		return []byte(null), nil
	}
	return json.Marshal(b.Bool)
}

// Scan implements sql.Scanner interface
func (b *Bool) Scan(input interface{}) error {
	switch val := input.(type) {
	case nil:
		b.Bool, b.Valid = false, false
	case bool:
		b.Bool, b.Valid = val, true
	case int64:
		b.Bool, b.Valid = val != 0, true
	case float64:
		b.Bool, b.Valid = val != 0.0, true
	case []byte:
		bb, err := strconv.ParseBool(string(val))
		b.Bool, b.Valid = bb, err == nil
	case string:
		bb, err := strconv.ParseBool(val)
		b.Bool, b.Valid = bb, err == nil
	default:
		return fmt.Errorf("failed to convert %v (%T) to null.Bool", input, input)
	}

	return nil
}

// Value implements driver.Valuer interface
func (b *Bool) Value() (driver.Value, error) {
	if !b.Valid {
		return nil, nil
	}
	return b.Bool, nil
}
