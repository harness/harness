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
	ID        int64            `json:"id"`
	CreatedBy int64            `json:"-"` // clients will use "reported_by"
	Created   int64            `json:"created"`
	Updated   int64            `json:"updated"`
	RepoID    int64            `json:"-"` // status checks are always returned for a commit in a repository
	CommitSHA string           `json:"-"` // status checks are always returned for a commit in a repository
	UID       string           `json:"uid"`
	Status    enum.CheckStatus `json:"status"`
	Summary   string           `json:"summary"`
	Link      string           `json:"link"`
	Metadata  json.RawMessage  `json:"metadata"`
	Started   int64            `json:"started"`
	Ended     int64            `json:"ended"`

	Payload    CheckPayload  `json:"payload"`
	ReportedBy PrincipalInfo `json:"reported_by"`
}

type CheckResult struct {
	UID    string           `json:"uid" db:"check_uid"`
	Status enum.CheckStatus `json:"status" db:"check_status"`
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

type ReqCheck struct {
	ID            int64  `json:"id"`
	CreatedBy     int64  `json:"-"` // clients will use "added_by"
	Created       int64  `json:"created"`
	RepoID        int64  `json:"-"` // required status checks are always returned for a single repository
	BranchPattern string `json:"branch_pattern"`
	CheckUID      string `json:"check_uid"`

	AddedBy PrincipalInfo `json:"added_by"`
}

type CheckPayloadText struct {
	Details string `json:"details"`
}

// CheckPayloadInternal is for internal use for more seamless integration for
// gitness CI status checks.
type CheckPayloadInternal struct {
	Number     int64 `json:"execution_number"`
	RepoID     int64 `json:"repo_id"`
	PipelineID int64 `json:"pipeline_id"`
}
