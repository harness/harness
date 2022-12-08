// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package null

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// String represents a string that may be null.
type String struct {
	sql.NullString
}

func NewString(s string) String {
	return String{
		sql.NullString{
			String: s,
			Valid:  true,
		},
	}
}

// FromNullableString returns a null String if the parameter is empty, a valid String otherwise.
func FromNullableString(s string) String {
	if s == "" {
		return String{}
	}
	return NewString(s)
}

// FromPtrString returns a null String if the parameter is nil, a valid String otherwise.
func FromPtrString(s *string) String {
	if s == nil {
		return String{}
	}
	return NewString(*s)
}

// ToString converts the null.String to a string.
func (s *String) ToString() string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// ToPtrString converts the null.String to a *string.
func (s *String) ToPtrString() *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *String) UnmarshalJSON(input []byte) error {
	var val interface{}
	if err := json.Unmarshal(input, &val); err != nil {
		return err
	}

	switch z := val.(type) {
	case string:
		s.String = z
		s.Valid = true
	default:
		s.String = ""
		s.Valid = false
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (s *String) MarshalJSON() ([]byte, error) {
	if !s.Valid {
		return []byte(null), nil
	}
	return json.Marshal(s.String)
}

// Scan implements sql.Scanner interface
func (s *String) Scan(input interface{}) error {
	switch val := input.(type) {
	case nil:
		s.String, s.Valid = "", false
	case []byte:
		s.String, s.Valid = string(val), true
	case string:
		s.String, s.Valid = val, true
	default:
		return fmt.Errorf("failed to convert %v (%T) to null.String", input, input)
	}

	return nil
}

// Value implements driver.Valuer interface
func (s *String) Value() (driver.Value, error) {
	if !s.Valid {
		return nil, nil
	}
	return s.String, nil
}
