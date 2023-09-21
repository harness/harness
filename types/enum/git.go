// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// TagSortOption specifies the available sort options for tags.
type TagSortOption int

const (
	TagSortOptionDefault TagSortOption = iota
	TagSortOptionName
	TagSortOptionDate
)

// ParseTagSortOption parses the tag sort option string
// and returns the equivalent enumeration.
func ParseTagSortOption(s string) TagSortOption {
	switch strings.ToLower(s) {
	case name:
		return TagSortOptionName
	case date:
		return TagSortOptionDate
	default:
		return TagSortOptionDefault
	}
}

// String returns a string representation of the tag sort option.
func (o TagSortOption) String() string {
	switch o {
	case TagSortOptionName:
		return name
	case TagSortOptionDate:
		return date
	case TagSortOptionDefault:
		return defaultString
	default:
		return undefined
	}
}
