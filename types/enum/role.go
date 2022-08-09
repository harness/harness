// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "encoding/json"

// Role defines the member role.
type Role int

// Role enumeration.
const (
	RoleDeveloper Role = iota
	RoleAdmin
)

// String returns the Role as a string.
func (e Role) String() string {
	switch e {
	case RoleDeveloper:
		return "developer"
	case RoleAdmin:
		return "admin"
	default:
		return "developer"
	}
}

// MarshalJSON marshals the Type as a JSON string.
func (e Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

// UnmarshalJSON unmashals a quoted json string to the enum value.
func (e *Role) UnmarshalJSON(b []byte) error {
	var v string
	json.Unmarshal(b, &v)
	switch v {
	case "admin":
		*e = RoleAdmin
	case "developer":
		*e = RoleDeveloper
	default:
		*e = RoleDeveloper
	}
	return nil
}
