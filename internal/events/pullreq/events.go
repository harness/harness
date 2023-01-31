// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

type Base struct {
	PullReqID    int64 `json:"pullreq_id"`
	SourceRepoID int64 `json:"source_repo_id"`
	TargetRepoID int64 `json:"repo_id"`
	PrincipalID  int64 `json:"principal_id"`
	Number       int64 `json:"number"`
}
