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
	SpaceAttrPathName
	SpaceAttrPath
	SpaceAttrName
	SpaceAttrCreated
	SpaceAttrUpdated
)

// ParseSpaceAttr parses the space attribute string
// and returns the equivalent enumeration.
func ParseSpaceAttr(s string) SpaceAttr {
	switch strings.ToLower(s) {
	case name:
		return SpaceAttrName
	case path:
		return SpaceAttrPath
	case pathName:
		return SpaceAttrPathName
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
	case SpaceAttrPathName:
		return pathName
	case SpaceAttrPath:
		return path
	case SpaceAttrName:
		return name
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
