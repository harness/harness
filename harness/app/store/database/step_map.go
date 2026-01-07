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
