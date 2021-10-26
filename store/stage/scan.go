// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stage

import (
	"database/sql"
	"encoding/json"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"

	"github.com/jmoiron/sqlx/types"
)

// helper function converts the Stage structure to a set
// of named query parameters.
func toParams(stage *core.Stage) map[string]interface{} {
	return map[string]interface{}{
		"stage_id":         stage.ID,
		"stage_repo_id":    stage.RepoID,
		"stage_build_id":   stage.BuildID,
		"stage_number":     stage.Number,
		"stage_name":       stage.Name,
		"stage_kind":       stage.Kind,
		"stage_type":       stage.Type,
		"stage_status":     stage.Status,
		"stage_error":      stage.Error,
		"stage_errignore":  stage.ErrIgnore,
		"stage_exit_code":  stage.ExitCode,
		"stage_limit":      stage.Limit,
		"stage_limit_repo": stage.LimitRepo,
		"stage_os":         stage.OS,
		"stage_arch":       stage.Arch,
		"stage_variant":    stage.Variant,
		"stage_kernel":     stage.Kernel,
		"stage_machine":    stage.Machine,
		"stage_started":    stage.Started,
		"stage_stopped":    stage.Stopped,
		"stage_created":    stage.Created,
		"stage_updated":    stage.Updated,
		"stage_version":    stage.Version,
		"stage_on_success": stage.OnSuccess,
		"stage_on_failure": stage.OnFailure,
		"stage_depends_on": encodeSlice(stage.DependsOn),
		"stage_labels":     encodeParams(stage.Labels),
	}
}

func encodeSlice(v []string) types.JSONText {
	raw, _ := json.Marshal(v)
	return types.JSONText(raw)
}

func encodeParams(v map[string]string) types.JSONText {
	raw, _ := json.Marshal(v)
	return types.JSONText(raw)
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRow(scanner db.Scanner, dest *core.Stage) error {
	depJSON := types.JSONText{}
	labJSON := types.JSONText{}
	err := scanner.Scan(
		&dest.ID,
		&dest.RepoID,
		&dest.BuildID,
		&dest.Number,
		&dest.Name,
		&dest.Kind,
		&dest.Type,
		&dest.Status,
		&dest.Error,
		&dest.ErrIgnore,
		&dest.ExitCode,
		&dest.Limit,
		&dest.LimitRepo,
		&dest.OS,
		&dest.Arch,
		&dest.Variant,
		&dest.Kernel,
		&dest.Machine,
		&dest.Started,
		&dest.Stopped,
		&dest.Created,
		&dest.Updated,
		&dest.Version,
		&dest.OnSuccess,
		&dest.OnFailure,
		&depJSON,
		&labJSON,
	)
	json.Unmarshal(depJSON, &dest.DependsOn)
	json.Unmarshal(labJSON, &dest.Labels)
	return err
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRowStep(scanner db.Scanner, stage *core.Stage, step *nullStep) error {
	depJSON := types.JSONText{}
	labJSON := types.JSONText{}
	stepDepJSON := types.JSONText{}
	err := scanner.Scan(
		&stage.ID,
		&stage.RepoID,
		&stage.BuildID,
		&stage.Number,
		&stage.Name,
		&stage.Kind,
		&stage.Type,
		&stage.Status,
		&stage.Error,
		&stage.ErrIgnore,
		&stage.ExitCode,
		&stage.Limit,
		&stage.LimitRepo,
		&stage.OS,
		&stage.Arch,
		&stage.Variant,
		&stage.Kernel,
		&stage.Machine,
		&stage.Started,
		&stage.Stopped,
		&stage.Created,
		&stage.Updated,
		&stage.Version,
		&stage.OnSuccess,
		&stage.OnFailure,
		&depJSON,
		&labJSON,
		&step.ID,
		&step.StageID,
		&step.Number,
		&step.Name,
		&step.Status,
		&step.Error,
		&step.ErrIgnore,
		&step.ExitCode,
		&step.Started,
		&step.Stopped,
		&step.Version,
		&stepDepJSON,
		&step.Image,
		&step.Detached,
		&step.Schema,
	)
	json.Unmarshal(depJSON, &stage.DependsOn)
	json.Unmarshal(labJSON, &stage.Labels)
	json.Unmarshal(stepDepJSON, &step.DependsOn)
	return err
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRows(rows *sql.Rows) ([]*core.Stage, error) {
	defer rows.Close()

	stages := []*core.Stage{}
	for rows.Next() {
		stage := new(core.Stage)
		err := scanRow(rows, stage)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRowsWithSteps(rows *sql.Rows) ([]*core.Stage, error) {
	defer rows.Close()

	stages := []*core.Stage{}
	var curr *core.Stage
	for rows.Next() {
		stage := new(core.Stage)
		step := new(nullStep)
		err := scanRowStep(rows, stage, step)
		if err != nil {
			return nil, err
		}
		if curr == nil || curr.ID != stage.ID {
			curr = stage
			stages = append(stages, stage)
		}
		if step.ID.Int64 != 0 {
			curr.Steps = append(curr.Steps, step.value())
		}
	}
	return stages, nil
}
