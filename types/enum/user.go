// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// UserAttr defines user attributes that can be
// used for sorting and filtering.
type UserAttr int

// Order enumeration.
const (
	UserAttrNone UserAttr = iota
	UserAttrUID
	UserAttrName
	UserAttrEmail
	UserAttrAdmin
	UserAttrCreated
	UserAttrUpdated
)

// ParseUserAttr parses the user attribute string
// and returns the equivalent enumeration.
func ParseUserAttr(s string) UserAttr {
	switch strings.ToLower(s) {
	case uid:
		return UserAttrUID
	case name:
		return UserAttrName
	case email:
		return UserAttrEmail
	case admin:
		return UserAttrAdmin
	case created, createdAt:
		return UserAttrCreated
	case updated, updatedAt:
		return UserAttrUpdated
	default:
		return UserAttrNone
	}
}
