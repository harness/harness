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

import "strings"

// UserAttr defines user attributes that can be
// used for sorting and filtering.
type UserAttr int

// Order enumeration.
const (
	UserAttrNone UserAttr = iota
	UserAttrUID
	UserAttrName
	UserAttrEmail
	UserAttrAdmin
	UserAttrCreated
	UserAttrUpdated
)

// ParseUserAttr parses the user attribute string
// and returns the equivalent enumeration.
func ParseUserAttr(s string) UserAttr {
	switch strings.ToLower(s) {
	case uid:
		return UserAttrUID
	case name:
		return UserAttrName
	case email:
		return UserAttrEmail
	case admin:
		return UserAttrAdmin
	case created, createdAt:
		return UserAttrCreated
	case updated, updatedAt:
		return UserAttrUpdated
	default:
		return UserAttrNone
	}
}
