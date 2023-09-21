// Copyright 2023 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

type Trigger struct {
	ID          int64                `json:"id"`
	Description string               `json:"description"`
	Type        string               `json:"trigger_type"`
	PipelineID  int64                `json:"pipeline_id"`
	Secret      string               `json:"-"`
	RepoID      int64                `json:"repo_id"`
	CreatedBy   int64                `json:"created_by"`
	Disabled    bool                 `json:"disabled"`
	Actions     []enum.TriggerAction `json:"actions"`
	UID         string               `json:"uid"`
	Created     int64                `json:"created"`
	Updated     int64                `json:"updated"`
	Version     int64                `json:"-"`
}
