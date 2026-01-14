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

package types

import "encoding/json"

const (
	PathSeparatorAsString = string(PathSeparator)
	PathSeparator         = '/'
)

// SpacePath represents a full path to a space.
type SpacePath struct {
	Value     string `json:"value"`
	IsPrimary bool   `json:"is_primary"`
	SpaceID   int64  `json:"space_id"`
}

// SpacePathSegment represents a segment of a path to a space.
type SpacePathSegment struct {
	// TODO: int64 ID doesn't match DB
	ID         int64  `json:"-"`
	Identifier string `json:"identifier"`
	IsPrimary  bool   `json:"is_primary"`
	SpaceID    int64  `json:"space_id"`
	ParentID   int64  `json:"parent_id"`
	CreatedBy  int64  `json:"created_by"`
	Created    int64  `json:"created"`
	Updated    int64  `json:"updated"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (s SpacePathSegment) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias SpacePathSegment
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(s),
		UID:   s.Identifier,
	})
}
