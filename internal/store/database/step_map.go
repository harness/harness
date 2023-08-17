// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types"
)

func mapInternalToStep(in *step) (*types.Step, error) {
	var dependsOn []string
	err := json.Unmarshal(in.DependsOn, &dependsOn)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal step.DependsOn: %w", err)
	}
	return &types.Step{
		ID:        in.ID,
		StageID:   in.StageID,
		Number:    in.Number,
		Name:      in.Name,
		Status:    in.Status,
		Error:     in.Error,
		ErrIgnore: in.ErrIgnore,
		ExitCode:  in.ExitCode,
		Started:   in.Started,
		Stopped:   in.Stopped,
		Version:   in.Version,
		DependsOn: dependsOn,
		Image:     in.Image,
		Detached:  in.Detached,
		Schema:    in.Schema,
	}, nil
}

func mapStepToInternal(in *types.Step) *step {
	return &step{
		ID:        in.ID,
		StageID:   in.StageID,
		Number:    in.Number,
		Name:      in.Name,
		Status:    in.Status,
		Error:     in.Error,
		ErrIgnore: in.ErrIgnore,
		ExitCode:  in.ExitCode,
		Started:   in.Started,
		Stopped:   in.Stopped,
		Version:   in.Version,
		DependsOn: EncodeToSQLXJSON(in.DependsOn),
		Image:     in.Image,
		Detached:  in.Detached,
		Schema:    in.Schema,
	}
}
