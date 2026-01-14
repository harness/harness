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

import (
	"encoding/json"

	"github.com/harness/gitness/types/enum"
)

type Template struct {
	ID          int64             `db:"template_id"              json:"-"`
	Description string            `db:"template_description"     json:"description"`
	Type        enum.ResolverType `db:"template_type"            json:"type"`
	SpaceID     int64             `db:"template_space_id"        json:"space_id"`
	Identifier  string            `db:"template_uid"             json:"identifier"`
	Data        string            `db:"template_data"            json:"data"`
	Created     int64             `db:"template_created"         json:"created"`
	Updated     int64             `db:"template_updated"         json:"updated"`
	Version     int64             `db:"template_version"         json:"-"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (t Template) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Template
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(t),
		UID:   t.Identifier,
	})
}
