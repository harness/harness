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

type AutoLink struct {
	ID        int64             `json:"id"`
	SpaceID   *int64            `json:"space_id,omitempty"`
	RepoID    *int64            `json:"repo_id,omitempty"`
	Type      enum.AutoLinkType `json:"type"`
	Pattern   string            `json:"pattern"`
	TargetURL string            `json:"target_url"`
	Created   int64             `json:"created"`
	Updated   int64             `json:"updated"`
	CreatedBy int64             `json:"created_by"`
	UpdatedBy int64             `json:"updated_by"`
}

type AutoLinkFilter struct {
	ListQueryFilter
	Inherited bool `json:"inherited,omitempty"`
}
