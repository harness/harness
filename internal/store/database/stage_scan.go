// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

func mapInternalToStage(in *stage) (*types.Stage, error) {
	var dependsOn []string
	err := json.Unmarshal(in.DependsOn, &dependsOn)
	if err != nil {
		fmt.Println("COULD NOT MAP: ", err)
		return nil, err
	}
	var labels map[string]string
	err = json.Unmarshal(in.Labels, &labels)
	if err != nil {
		return nil, err
	}
	return &types.Stage{
		ID:          in.ID,
		ExecutionID: in.ExecutionID,
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
		DependsOn:   EncodeToJSON(in.DependsOn),
		Labels:      EncodeToJSON(in.Labels),
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
		step := new(types.Step)
		err := scanRowStep(rows, stage, step)
		if err != nil {
			return nil, err
		}
		if curr == nil || curr.ID != stage.ID {
			curr = stage
			stages = append(stages, curr)
		}
		if step.ID != 0 {
			curr.Steps = append(curr.Steps, step)
		}
	}
	return stages, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRowStep(rows *sql.Rows, stage *types.Stage, step *types.Step) error {
	depJSON := sqlxtypes.JSONText{}
	labJSON := sqlxtypes.JSONText{}
	stepDepJSON := sqlxtypes.JSONText{}
	err := rows.Scan(
		&stage.ID,
		&stage.ExecutionID,
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
	json.Unmarshal(depJSON, &stage.DependsOn)
	json.Unmarshal(labJSON, &stage.Labels)
	json.Unmarshal(stepDepJSON, &step.DependsOn)
	return err
}
