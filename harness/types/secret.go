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

type Secret struct {
	ID          int64  `db:"secret_id"              json:"-"`
	Description string `db:"secret_description"     json:"description"`
	SpaceID     int64  `db:"secret_space_id"        json:"space_id"`
	CreatedBy   int64  `db:"secret_created_by"      json:"created_by"`
	Identifier  string `db:"secret_uid"             json:"identifier"`
	Data        string `db:"secret_data"            json:"-"`
	Created     int64  `db:"secret_created"         json:"created"`
	Updated     int64  `db:"secret_updated"         json:"updated"`
	Version     int64  `db:"secret_version"         json:"-"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (s Secret) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Secret
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(s),
		UID:   s.Identifier,
	})
}

// Copy makes a copy of the secret without the value.
func (s *Secret) CopyWithoutData() *Secret {
	return &Secret{
		ID:          s.ID,
		Description: s.Description,
		Identifier:  s.Identifier,
		SpaceID:     s.SpaceID,
		Created:     s.Created,
		Updated:     s.Updated,
		Version:     s.Version,
	}
}
