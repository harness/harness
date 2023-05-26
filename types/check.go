// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	Type      string           `json:"type"`
	UID       string           `json:"uid"`
	Status    enum.CheckStatus `json:"status"`
	Summary   string           `json:"summary"`
	Link      string           `json:"link"`
	Payload   json.RawMessage  `json:"payload"`
	Metadata  json.RawMessage  `json:"metadata"`

	ReportedBy PrincipalInfo `json:"reported_by"`
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
