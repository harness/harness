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
	"github.com/harness/gitness/types/enum"
)

// RepoActivity represents an activity entry for a repository.
type RepoActivity struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Key         string `json:"-"`

	Type      enum.RepoActivityType `json:"type"`
	Payload   RepoActivityPayload   `json:"payload,omitempty"`
	Timestamp int64                 `json:"timestamp"`
}

// RepoActivityFilter stores repository activity query parameters.
type RepoActivityFilter struct {
	After  int64 `json:"after"`
	Before int64 `json:"before"`
	Page   int   `json:"page"`
	Size   int   `json:"size"`
}
