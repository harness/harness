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

type GitspaceEvent struct {
	ID         int64                   `json:"-"`
	Event      enum.GitspaceEventType  `json:"event,omitempty"`
	EntityID   int64                   `json:"-"`
	QueryKey   string                  `json:"query_key,omitempty"`
	EntityType enum.GitspaceEntityType `json:"entity_type,omitempty"`
	Timestamp  int64                   `json:"timestamp,omitempty"`
	Created    int64                   `json:"created,omitempty"`
}

type GitspaceEventResponse struct {
	GitspaceEvent
	EventTime string `json:"event_time,omitempty"`
	Message   string `json:"message,omitempty"`
}

type GitspaceEventFilter struct {
	Pagination
	QueryKey   string
	EntityID   int64
	EntityType enum.GitspaceEntityType
	SkipEvents []enum.GitspaceEventType // not include events
}
