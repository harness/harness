// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import (
	"strings"
)

// MembershipUserSort represents membership user sort order.
type MembershipUserSort string

// MembershipUserSort enumeration.
const (
	MembershipUserSortName    MembershipUserSort = name
	MembershipUserSortCreated MembershipUserSort = created
)

var membershipUserSorts = sortEnum([]MembershipUserSort{
	MembershipUserSortName,
	MembershipUserSortCreated,
})

func (MembershipUserSort) Enum() []interface{} { return toInterfaceSlice(membershipUserSorts) }
func (s MembershipUserSort) Sanitize() (MembershipUserSort, bool) {
	return Sanitize(s, GetAllMembershipUserSorts)
}
func GetAllMembershipUserSorts() ([]MembershipUserSort, MembershipUserSort) {
	return membershipUserSorts, MembershipUserSortName
}

// ParseMembershipUserSort parses the membership user sort attribute string
// and returns the equivalent enumeration.
func ParseMembershipUserSort(s string) MembershipUserSort {
	switch strings.ToLower(s) {
	case name:
		return MembershipUserSortName
	case created, createdAt:
		return MembershipUserSortCreated
	default:
		return MembershipUserSortName
	}
}

// String returns the string representation of the attribute.
func (s MembershipUserSort) String() string {
	switch s {
	case MembershipUserSortName:
		return name
	case MembershipUserSortCreated:
		return created
	default:
		return undefined
	}
}

// MembershipSpaceSort represents membership space sort order.
type MembershipSpaceSort string

// MembershipSpaceSort enumeration.
const (
	MembershipSpaceSortUID     MembershipSpaceSort = uid
	MembershipSpaceSortCreated MembershipSpaceSort = created
)

var membershipSpaceSorts = sortEnum([]MembershipSpaceSort{
	MembershipSpaceSortUID,
	MembershipSpaceSortCreated,
})

func (MembershipSpaceSort) Enum() []interface{} { return toInterfaceSlice(membershipSpaceSorts) }
func (s MembershipSpaceSort) Sanitize() (MembershipSpaceSort, bool) {
	return Sanitize(s, GetAllMembershipSpaceSorts)
}
func GetAllMembershipSpaceSorts() ([]MembershipSpaceSort, MembershipSpaceSort) {
	return membershipSpaceSorts, MembershipSpaceSortUID
}

// ParseMembershipSpaceSort parses the membership space sort attribute string
// and returns the equivalent enumeration.
func ParseMembershipSpaceSort(s string) MembershipSpaceSort {
	switch strings.ToLower(s) {
	case name:
		return MembershipSpaceSortUID
	case created, createdAt:
		return MembershipSpaceSortCreated
	default:
		return MembershipSpaceSortUID
	}
}

// String returns the string representation of the attribute.
func (s MembershipSpaceSort) String() string {
	switch s {
	case MembershipSpaceSortUID:
		return uid
	case MembershipSpaceSortCreated:
		return created
	default:
		return undefined
	}
}
