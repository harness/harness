// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import (
	"strings"
)

// MembershipSort represents membership sort order.
type MembershipSort int

// Order enumeration.
const (
	MembershipSortNone MembershipSort = iota
	MembershipSortName
	MembershipSortCreated
)

// ParseMembershipSort parses the membership sort attribute string
// and returns the equivalent enumeration.
func ParseMembershipSort(s string) MembershipSort {
	switch strings.ToLower(s) {
	case name:
		return MembershipSortName
	case created, createdAt:
		return MembershipSortCreated
	default:
		return MembershipSortNone
	}
}

// String returns the string representation of the attribute.
func (a MembershipSort) String() string {
	switch a {
	case MembershipSortName:
		return name
	case MembershipSortCreated:
		return created
	case MembershipSortNone:
		return ""
	default:
		return undefined
	}
}
