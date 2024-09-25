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

type Check struct {
	ID         int64            `json:"id"`
	CreatedBy  int64            `json:"-"` // clients will use "reported_by"
	Created    int64            `json:"created,omitempty"`
	Updated    int64            `json:"updated,omitempty"`
	RepoID     int64            `json:"-"` // status checks are always returned for a commit in a repository
	CommitSHA  string           `json:"-"` // status checks are always returned for a commit in a repository
	Identifier string           `json:"identifier"`
	Status     enum.CheckStatus `json:"status"`
	Summary    string           `json:"summary,omitempty"`
	Link       string           `json:"link,omitempty"`
	Metadata   json.RawMessage  `json:"metadata"`
	Started    int64            `json:"started,omitempty"`
	Ended      int64            `json:"ended,omitempty"`

	Payload    CheckPayload   `json:"payload"`
	ReportedBy *PrincipalInfo `json:"reported_by,omitempty"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (c Check) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Check
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(c),
		UID:   c.Identifier,
	})
}

type CheckResult struct {
	Identifier string           `json:"identifier" db:"check_uid"`
	Status     enum.CheckStatus `json:"status" db:"check_status"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (s CheckResult) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias CheckResult
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(s),
		UID:   s.Identifier,
	})
}

type CheckPayload struct {
	Version string                `json:"version"`
	Kind    enum.CheckPayloadKind `json:"kind"`
	Data    json.RawMessage       `json:"data"`
}

// CheckListOptions holds list status checks query parameters.
type CheckListOptions struct {
	ListQueryFilter
}

// CheckRecentOptions holds list recent status check query parameters.
type CheckRecentOptions struct {
	Query string
	Since int64
}

type CheckPayloadText struct {
	Details string `json:"details"`
}

// CheckPayloadInternal is for internal use for more seamless integration for
// Harness CI status checks.
type CheckPayloadInternal struct {
	Number     int64 `json:"execution_number"`
	RepoID     int64 `json:"repo_id"`
	PipelineID int64 `json:"pipeline_id"`
}

type PullReqChecks struct {
	CommitSHA string         `json:"commit_sha"`
	Checks    []PullReqCheck `json:"checks"`
}

type PullReqCheck struct {
	Required   bool  `json:"required"`
	Bypassable bool  `json:"bypassable"`
	Check      Check `json:"check"`
}
