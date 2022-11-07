// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import (
	"strings"
)

// Defines repo attributes that can be used for sorting and filtering.
type RepoAttr int

// Order enumeration.
const (
	RepoAttrNone RepoAttr = iota
	RepoAttrPath
	RepoAttrUID
	RepoAttrCreated
	RepoAttrUpdated
)

// ParseRepoAtrr parses the repo attribute string
// and returns the equivalent enumeration.
func ParseRepoAtrr(s string) RepoAttr {
	switch strings.ToLower(s) {
	case uid:
		return RepoAttrUID
	case path:
		return RepoAttrPath
	case created, createdAt:
		return RepoAttrCreated
	case updated, updatedAt:
		return RepoAttrUpdated
	default:
		return RepoAttrNone
	}
}

// String returns the string representation of the attribute.
func (a RepoAttr) String() string {
	switch a {
	case RepoAttrPath:
		return path
	case RepoAttrUID:
		return uid
	case RepoAttrCreated:
		return created
	case RepoAttrUpdated:
		return updated
	case RepoAttrNone:
		return ""
	default:
		return undefined
	}
}
