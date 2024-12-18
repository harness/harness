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

package manager

import (
	"context"
	"errors"
	"time"

	"github.com/harness/gitness/app/pipeline/checks"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type setup struct {
	Executions  store.ExecutionStore
	Checks      store.CheckStore
	SSEStreamer sse.Streamer
	Pipelines   store.PipelineStore
	Repos       store.RepoStore
	Steps       store.StepStore
	Stages      store.StageStore
	Users       store.PrincipalStore
}

func (s *setup) do(ctx context.Context, stage *types.Stage) error {
	execution, err := s.Executions.Find(noContext, stage.ExecutionID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find the execution")
		return err
	}

	log := log.With().
		Int64("execution.number", execution.Number).
		Int64("execution.id", execution.ID).
		Int64("stage.id", stage.ID).
		Int64("repo.id", execution.RepoID).
		Logger()

	repo, err := s.Repos.Find(noContext, execution.RepoID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find the repository")
		return err
	}

	if len(stage.Error) > 500 {
		stage.Error = stage.Error[:500]
	}
	err = s.Stages.Update(noContext, stage)
	if err != nil {
		log.Error().Err(err).
			Str("stage.status", string(stage.Status)).
			Msg("manager: cannot update the stage")
		return err
	}

	// TODO: create all the steps as part of a single transaction?
	for _, step := range stage.Steps {
		if len(step.Error) > 500 {
			step.Error = step.Error[:500]
		}
		err := s.Steps.Create(noContext, step)
		if err != nil {
			log.Error().Err(err).
				Str("stage.status", string(stage.Status)).
				Str("step.name", step.Name).
				Int64("step.id", step.ID).
				Msg("manager: cannot persist the step")
			return err
		}
	}

	_, err = s.updateExecution(noContext, execution)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot update the execution")
		return err
	}
	pipeline, err := s.Pipelines.Find(ctx, execution.PipelineID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find pipeline")
		return err
	}
	// try to write to the checks store - if not, log an error and continue
	err = checks.Write(ctx, s.Checks, execution, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("manager: could not write to checks store")
	}
	stages, err := s.Stages.ListWithSteps(noContext, execution.ID)
	if err != nil {
		log.Error().Err(err).Msg("manager: could not list stages with steps")
		return err
	}
	execution.Stages = stages

	s.SSEStreamer.Publish(noContext, repo.ParentID, enum.SSETypeExecutionRunning, execution)

	return nil
}

// helper function that updates the execution status from pending to running.
// This accounts for the fact that another agent may have already updated
// the execution status, which may happen if two stages execute concurrently.
func (s *setup) updateExecution(ctx context.Context, execution *types.Execution) (bool, error) {
	if execution.Status != enum.CIStatusPending {
		return false, nil
	}
	execution.Started = time.Now().UnixMilli()
	execution.Status = enum.CIStatusRunning
	err := s.Executions.Update(ctx, execution)
	if errors.Is(err, gitness_store.ErrVersionConflict) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
