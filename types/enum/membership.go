// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import (
	"strings"
)

// MembershipSort represents membership sort order.
type MembershipSort string

// Order enumeration.
const (
	MembershipSortName    = name
	MembershipSortCreated = created
)

var membershipSorts = sortEnum([]MembershipSort{
	MembershipSortName,
	MembershipSortCreated,
})

func (MembershipSort) Enum() []interface{}                { return toInterfaceSlice(membershipSorts) }
func (s MembershipSort) Sanitize() (MembershipSort, bool) { return Sanitize(s, GetAllMembershipSorts) }
func GetAllMembershipSorts() ([]MembershipSort, MembershipSort) {
	return membershipSorts, MembershipSortName
}

// ParseMembershipSort parses the membership sort attribute string
// and returns the equivalent enumeration.
func ParseMembershipSort(s string) MembershipSort {
	switch strings.ToLower(s) {
	case name:
		return MembershipSortName
	case created, createdAt:
		return MembershipSortCreated
	default:
		return MembershipSortName
	}
}

// String returns the string representation of the attribute.
func (s MembershipSort) String() string {
	switch s {
	case MembershipSortName:
		return name
	case MembershipSortCreated:
		return created
	default:
		return undefined
	}
}
