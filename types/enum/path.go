// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// PathTargetType defines the type of the target of a path.
type PathTargetType string

const (
	PathTargetTypeRepo  PathTargetType = "repo"
	PathTargetTypeSpace PathTargetType = "space"
)

// TODO: Should we replace Path.IsAlias with a Path.Type property? Unless needed, bool would be more efficient
// // Defines the type of a path
// type PathType string

// const (
// 	// Path is only an alias - it doesn't dictate where the target is actually residing.
// 	PathTypeAlias PathTargetType = "alias"

// 	// Path is representing the residency of a resource (e.g. chain of parent spaces)
// 	PathTypePrimary PathTargetType = "primary"
// )

// PathAttr defines path attributes that can be used for sorting and filtering.
type PathAttr int

// Order enumeration.
const (
	PathAttrNone PathAttr = iota
	PathAttrPath
	PathAttrCreated
	PathAttrUpdated
)

// ParsePathAttr parses the path attribute string
// and returns the equivalent enumeration.
func ParsePathAttr(s string) PathAttr {
	switch strings.ToLower(s) {
	case path:
		return PathAttrPath
	case created, createdAt:
		return PathAttrCreated
	case updated, updatedAt:
		return PathAttrUpdated
	default:
		return PathAttrNone
	}
}

// String returns the string representation of the attribute.
func (a PathAttr) String() string {
	switch a {
	case PathAttrPath:
		return path
	case PathAttrCreated:
		return created
	case PathAttrUpdated:
		return updated
	case PathAttrNone:
		return ""
	default:
		return undefined
	}
}
