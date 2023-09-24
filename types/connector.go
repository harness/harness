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

type Connector struct {
	ID          int64  `db:"connector_id"              json:"id"`
	Description string `db:"connector_description"     json:"description"`
	SpaceID     int64  `db:"connector_space_id"        json:"space_id"`
	UID         string `db:"connector_uid"             json:"uid"`
	Type        string `db:"connector_type"            json:"type"`
	Data        string `db:"connector_data"            json:"data"`
	Created     int64  `db:"connector_created"         json:"created"`
	Updated     int64  `db:"connector_updated"         json:"updated"`
	Version     int64  `db:"connector_version"         json:"-"`
}
