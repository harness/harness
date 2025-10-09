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

package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
)

type nullstep struct {
	ID            sql.NullInt64      `db:"step_id"`
	StageID       sql.NullInt64      `db:"step_stage_id"`
	Number        sql.NullInt64      `db:"step_number"`
	ParentGroupID sql.NullInt64      `db:"step_parent_group_id"`
	Name          sql.NullString     `db:"step_name"`
	Status        sql.NullString     `db:"step_status"`
	Error         sql.NullString     `db:"step_error"`
	ErrIgnore     sql.NullBool       `db:"step_errignore"`
	ExitCode      sql.NullInt64      `db:"step_exit_code"`
	Started       sql.NullInt64      `db:"step_started"`
	Stopped       sql.NullInt64      `db:"step_stopped"`
	Version       sql.NullInt64      `db:"step_version"`
	DependsOn     sqlxtypes.JSONText `db:"step_depends_on"`
	Image         sql.NullString     `db:"step_image"`
	Detached      sql.NullBool       `db:"step_detached"`
	Schema        sql.NullString     `db:"step_schema"`
}

// used for join operations where fields may be null.
func convertFromNullStep(nullstep *nullstep) (*types.Step, error) {
	var dependsOn []string
	err := json.Unmarshal(nullstep.DependsOn, &dependsOn)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal step.depends_on: %w", err)
	}
	return &types.Step{
		ID:        nullstep.ID.Int64,
		StageID:   nullstep.StageID.Int64,
		Number:    nullstep.Number.Int64,
		Name:      nullstep.Name.String,
		Status:    enum.ParseCIStatus(nullstep.Status.String),
		Error:     nullstep.Error.String,
		ErrIgnore: nullstep.ErrIgnore.Bool,
		ExitCode:  int(nullstep.ExitCode.Int64),
		Started:   nullstep.Started.Int64,
		Stopped:   nullstep.Stopped.Int64,
		Version:   nullstep.Version.Int64,
		DependsOn: dependsOn,
		Image:     nullstep.Image.String,
		Detached:  nullstep.Detached.Bool,
		Schema:    nullstep.Schema.String,
	}, nil
}

func mapInternalToStage(in *stage) (*types.Stage, error) {
	var dependsOn []string
	err := json.Unmarshal(in.DependsOn, &dependsOn)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal stage.depends_on")
	}
	var labels map[string]string
	err = json.Unmarshal(in.Labels, &labels)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal stage.labels")
	}
	return &types.Stage{
		ID:          in.ID,
		ExecutionID: in.ExecutionID,
		RepoID:      in.RepoID,
		Number:      in.Number,
		Name:        in.Name,
		Kind:        in.Kind,
		Type:        in.Type,
		Status:      in.Status,
		Error:       in.Error,
		ErrIgnore:   in.ErrIgnore,
		ExitCode:    in.ExitCode,
		Machine:     in.Machine,
		OS:          in.OS,
		Arch:        in.Arch,
		Variant:     in.Variant,
		Kernel:      in.Kernel,
		Limit:       in.Limit,
		LimitRepo:   in.LimitRepo,
		Started:     in.Started,
		Stopped:     in.Stopped,
		Created:     in.Created,
		Updated:     in.Updated,
		Version:     in.Version,
		OnSuccess:   in.OnSuccess,
		OnFailure:   in.OnFailure,
		DependsOn:   dependsOn,
		Labels:      labels,
	}, nil
}

func mapStageToInternal(in *types.Stage) *stage {
	return &stage{
		ID:          in.ID,
		ExecutionID: in.ExecutionID,
		RepoID:      in.RepoID,
		Number:      in.Number,
		Name:        in.Name,
		Kind:        in.Kind,
		Type:        in.Type,
		Status:      in.Status,
		Error:       in.Error,
		ErrIgnore:   in.ErrIgnore,
		ExitCode:    in.ExitCode,
		Machine:     in.Machine,
		OS:          in.OS,
		Arch:        in.Arch,
		Variant:     in.Variant,
		Kernel:      in.Kernel,
		Limit:       in.Limit,
		LimitRepo:   in.LimitRepo,
		Started:     in.Started,
		Stopped:     in.Stopped,
		Created:     in.Created,
		Updated:     in.Updated,
		Version:     in.Version,
		OnSuccess:   in.OnSuccess,
		OnFailure:   in.OnFailure,
		DependsOn:   EncodeToSQLXJSON(in.DependsOn),
		Labels:      EncodeToSQLXJSON(in.Labels),
	}
}

func mapInternalToStageList(in []*stage) ([]*types.Stage, error) {
	stages := make([]*types.Stage, len(in))
	for i, k := range in {
		s, err := mapInternalToStage(k)
		if err != nil {
			return nil, err
		}
		stages[i] = s
	}
	return stages, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRowsWithSteps(rows *sql.Rows) ([]*types.Stage, error) {
	defer rows.Close()

	stages := []*types.Stage{}
	var curr *types.Stage
	for rows.Next() {
		stage := new(types.Stage)
		step := new(nullstep)
		err := scanRowStep(rows, stage, step)
		if err != nil {
			return nil, err
		}
		if curr == nil || curr.ID != stage.ID {
			curr = stage
			stages = append(stages, curr)
		}
		if step.ID.Valid {
			convertedStep, err := convertFromNullStep(step)
			if err != nil {
				return nil, err
			}
			curr.Steps = append(curr.Steps, convertedStep)
		}
	}
	return stages, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRowStep(rows *sql.Rows, stage *types.Stage, step *nullstep) error {
	depJSON := sqlxtypes.JSONText{}
	labJSON := sqlxtypes.JSONText{}
	stepDepJSON := sqlxtypes.JSONText{}
	err := rows.Scan(
		&stage.ID,
		&stage.ExecutionID,
		&stage.RepoID,
		&stage.Number,
		&stage.Name,
		&stage.Kind,
		&stage.Type,
		&stage.Status,
		&stage.Error,
		&stage.ErrIgnore,
		&stage.ExitCode,
		&stage.Machine,
		&stage.OS,
		&stage.Arch,
		&stage.Variant,
		&stage.Kernel,
		&stage.Limit,
		&stage.LimitRepo,
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
	if err != nil {
		return fmt.Errorf("failed to scan row: %w", err)
	}
	err = json.Unmarshal(depJSON, &stage.DependsOn)
	if err != nil {
		return fmt.Errorf("failed to unmarshal depJSON: %w", err)
	}
	err = json.Unmarshal(labJSON, &stage.Labels)
	if err != nil {
		return fmt.Errorf("failed to unmarshal labJSON: %w", err)
	}
	if step.ID.Valid {
		// try to unmarshal step dependencies if step exists
		err = json.Unmarshal(stepDepJSON, &step.DependsOn)
		if err != nil {
			return fmt.Errorf("failed to unmarshal stepDepJSON: %w", err)
		}
	}

	return nil
}
