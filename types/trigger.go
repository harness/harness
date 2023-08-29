// Copyright 2023 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Trigger struct {
	ID          int64  `db:"trigger_id"              json:"id"`
	Description string `db:"trigger_description"     json:"description"`
	PipelineID  int64  `db:"trigger_pipeline_id"     json:"pipeline_id"`
	UID         string `db:"trigger_uid"             json:"uid"`
	Created     int64  `db:"trigger_created"         json:"created"`
	Updated     int64  `db:"trigger_updated"         json:"updated"`
	Version     int64  `db:"trigger_version"         json:"-"`
}
