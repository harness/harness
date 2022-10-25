// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// BranchSortOption specifies the available sort options for branches.
type BranchSortOption int

const (
	BranchSortOptionDefault BranchSortOption = iota
	BranchSortOptionName
	BranchSortOptionDate
)

// ParseBranchSortOption parses the branch sort option string
// and returns the equivalent enumeration.
func ParseBranchSortOption(s string) BranchSortOption {
	switch strings.ToLower(s) {
	case name:
		return BranchSortOptionName
	case date:
		return BranchSortOptionDate
	default:
		return BranchSortOptionDefault
	}
}

// String returns a string representation of the branch sort option.
func (o BranchSortOption) String() string {
	switch o {
	case BranchSortOptionName:
		return name
	case BranchSortOptionDate:
		return date
	case BranchSortOptionDefault:
		return defaultString
	default:
		return undefined
	}
}
