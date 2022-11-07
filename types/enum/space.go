// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// SpaceAttr defines space attributes that can be used for sorting and filtering.
type SpaceAttr int

// Order enumeration.
const (
	SpaceAttrNone SpaceAttr = iota
	SpaceAttrPath
	SpaceAttrUID
	SpaceAttrCreated
	SpaceAttrUpdated
)

// ParseSpaceAttr parses the space attribute string
// and returns the equivalent enumeration.
func ParseSpaceAttr(s string) SpaceAttr {
	switch strings.ToLower(s) {
	case uid:
		return SpaceAttrUID
	case path:
		return SpaceAttrPath
	case created, createdAt:
		return SpaceAttrCreated
	case updated, updatedAt:
		return SpaceAttrUpdated
	default:
		return SpaceAttrNone
	}
}

// String returns the string representation of the attribute.
func (a SpaceAttr) String() string {
	switch a {
	case SpaceAttrPath:
		return path
	case SpaceAttrUID:
		return uid
	case SpaceAttrCreated:
		return created
	case SpaceAttrUpdated:
		return updated
	case SpaceAttrNone:
		return ""
	default:
		return undefined
	}
}
