// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// Defines repo attributes that can be used for sorting and filtering.
type RepoAttr int

// Order enumeration.
const (
	RepoAttrNone RepoAttr = iota
	RepoAttrId
	RepoAttrName
	RepoAttrFqn
	RepoAttrDisplayName
	RepoAttrCreated
	RepoAttrUpdated
)

// ParseRepoAtrr parses the user attribute string
// and returns the equivalent enumeration.
func ParseRepoAtrr(s string) RepoAttr {
	switch strings.ToLower(s) {
	case "id":
		return RepoAttrId
	case "name":
		return RepoAttrName
	case "fqn":
		return RepoAttrFqn
	case "displayName":
		return RepoAttrDisplayName
	case "created", "created_at":
		return RepoAttrCreated
	case "updated", "updated_at":
		return RepoAttrUpdated
	default:
		return RepoAttrNone
	}
}
