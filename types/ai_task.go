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

import "github.com/harness/gitness/types/enum"

type AITask struct {
	ID                 int64            `json:"id"`
	Identifier         string           `json:"identifier"`
	GitspaceConfigID   int64            `json:"-"`
	GitspaceInstanceID int64            `json:"-"`
	GitspaceConfig     *GitspaceConfig  `json:"gitspace_config"`
	InitialPrompt      string           `json:"initial_prompt"`
	DisplayName        string           `json:"display_name"`
	UserUID            string           `json:"user_uid"`
	SpaceID            int64            `json:"space_id"`
	Created            int64            `json:"created"`
	Updated            int64            `json:"updated"`
	APIURL             *string          `json:"api_url,omitempty"`
	AIAgent            enum.AIAgent     `json:"ai_agent"`
	State              enum.AITaskState `json:"state"`
	Output             *string          `json:"output,omitempty"`
}
type AITaskFilter struct {
	QueryFilter    ListQueryFilter
	SpaceID        int64
	UserIdentifier string
	AIAgents       []enum.AIAgent
	States         []enum.AITaskState
}
