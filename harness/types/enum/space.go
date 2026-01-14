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

// SpaceAttr defines space attributes that can be used for sorting and filtering.
type SpaceAttr int

// Order enumeration.
const (
	SpaceAttrNone SpaceAttr = iota
	// TODO [CODE-1363]: remove after identifier migration.
	SpaceAttrUID
	SpaceAttrIdentifier
	SpaceAttrCreated
	SpaceAttrUpdated
	SpaceAttrDeleted
)

// ParseSpaceAttr parses the space attribute string
// and returns the equivalent enumeration.
func ParseSpaceAttr(s string) SpaceAttr {
	switch strings.ToLower(s) {
	// TODO [CODE-1363]: remove after identifier migration.
	case uid:
		return SpaceAttrUID
	case identifier:
		return SpaceAttrIdentifier
	case created, createdAt:
		return SpaceAttrCreated
	case updated, updatedAt:
		return SpaceAttrUpdated
	case deleted, deletedAt:
		return SpaceAttrDeleted
	default:
		return SpaceAttrNone
	}
}

// String returns the string representation of the attribute.
func (a SpaceAttr) String() string {
	switch a {
	// TODO [CODE-1363]: remove after identifier migration.
	case SpaceAttrUID:
		return uid
	case SpaceAttrIdentifier:
		return identifier
	case SpaceAttrCreated:
		return created
	case SpaceAttrUpdated:
		return updated
	case SpaceAttrDeleted:
		return deleted
	case SpaceAttrNone:
		return ""
	default:
		return undefined
	}
}
