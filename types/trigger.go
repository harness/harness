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

type Trigger struct {
	ID          int64                `json:"-"`
	Description string               `json:"description"`
	Type        string               `json:"trigger_type"`
	PipelineID  int64                `json:"pipeline_id"`
	Secret      string               `json:"-"`
	RepoID      int64                `json:"repo_id"`
	CreatedBy   int64                `json:"created_by"`
	Disabled    bool                 `json:"disabled"`
	Actions     []enum.TriggerAction `json:"actions"`
	Identifier  string               `json:"identifier"`
	Created     int64                `json:"created"`
	Updated     int64                `json:"updated"`
	Version     int64                `json:"-"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (s Trigger) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Trigger
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(s),
		UID:   s.Identifier,
	})
}
