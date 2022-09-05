// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// Defines space attributes that can be used for sorting and filtering.
type SpaceAttr int

// Order enumeration.
const (
	SpaceAttrNone SpaceAttr = iota
	SpaceAttrId
	SpaceAttrName
	SpaceAttrFqn
	SpaceAttrDisplayName
	SpaceAttrCreated
	SpaceAttrUpdated
)

// ParseSpaceAttr parses the user attribute string
// and returns the equivalent enumeration.
func ParseSpaceAttr(s string) SpaceAttr {
	switch strings.ToLower(s) {
	case "id":
		return SpaceAttrId
	case "name":
		return SpaceAttrName
	case "fqn":
		return SpaceAttrFqn
	case "displayName":
		return SpaceAttrDisplayName
	case "created", "created_at":
		return SpaceAttrCreated
	case "updated", "updated_at":
		return SpaceAttrUpdated
	default:
		return SpaceAttrNone
	}
}
