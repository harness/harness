// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

type Pipeline struct {
	ID            int64        `db:"pipeline_id"              json:"id"`
	Description   string       `db:"pipeline_description"     json:"description"`
	ParentID      int64        `db:"pipeline_parent_id"       json:"parent_id"` // ID of the parent space
	UID           string       `db:"pipeline_uid"             json:"uid"`
	Seq           int64        `db:"pipeline_seq"             json:"seq"`     // last execution number for this pipeline
	RepoID        int64        `db:"pipeline_repo_id"         json:"repo_id"` // null if repo_type != gitness
	RepoType      enum.ScmType `db:"pipeline_repo_type"       json:"repo_type"`
	RepoName      string       `db:"pipeline_repo_name"       json:"repo_name"`
	DefaultBranch string       `db:"pipeline_default_branch"  json:"default_branch"`
	ConfigPath    string       `db:"pipeline_config_path"     json:"config_path"`
	Created       int64        `db:"pipeline_created"         json:"created"`
	Updated       int64        `db:"pipeline_updated"         json:"updated"`
	Version       int64        `db:"pipeline_version"         json:"version"`
}

// PipelineFilter stores pipeline query parameters.
type PipelineFilter struct {
	Page  int    `json:"page"`
	Size  int    `json:"size"`
	Query string `json:"query"`
}
