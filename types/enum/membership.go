// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
