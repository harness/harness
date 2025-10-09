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

package canceler

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type service struct {
	executionStore store.ExecutionStore
	sseStreamer    sse.Streamer
	repoStore      store.RepoStore
	scheduler      scheduler.Scheduler
	stageStore     store.StageStore
	stepStore      store.StepStore
}

// Canceler cancels a build.
type Canceler interface {
	// Cancel cancels the provided execution.
	Cancel(ctx context.Context, repo *types.RepositoryCore, execution *types.Execution) error
}

// New returns a cancellation service that encapsulates
// all cancellation operations.
func New(
	executionStore store.ExecutionStore,
	sseStreamer sse.Streamer,
	repoStore store.RepoStore,
	scheduler scheduler.Scheduler,
	stageStore store.StageStore,
	stepStore store.StepStore,
) Canceler {
	return &service{
		executionStore: executionStore,
		sseStreamer:    sseStreamer,
		repoStore:      repoStore,
		scheduler:      scheduler,
		stageStore:     stageStore,
		stepStore:      stepStore,
	}
}

//nolint:gocognit // refactor if needed.
func (s *service) Cancel(ctx context.Context, repo *types.RepositoryCore, execution *types.Execution) error {
	log := log.With().
		Int64("execution.id", execution.ID).
		Str("execution.status", string(execution.Status)).
		Str("execution.Ref", execution.Ref).
		Logger()

	// do not cancel the build if the build status is
	// complete. only cancel the build if the status is
	// running or pending.
	if execution.Status != enum.CIStatusPending &&
		execution.Status != enum.CIStatusRunning {
		return nil
	}

	// update the build status to killed. if the update fails
	// due to an optimistic lock error it means the build has
	// already started, and should now be ignored.
	now := time.Now().UnixMilli()
	execution.Status = enum.CIStatusKilled
	execution.Finished = now
	if execution.Started == 0 {
		execution.Started = now
	}

	err := s.executionStore.Update(ctx, execution)
	if err != nil {
		return fmt.Errorf("could not update execution status to canceled: %w", err)
	}

	stages, err := s.stageStore.ListWithSteps(ctx, execution.ID)
	if err != nil {
		return fmt.Errorf("could not list stages with steps: %w", err)
	}

	// update the status of all steps to indicate they
	// were killed or skipped.
	for _, stage := range stages {
		if stage.Status.IsDone() {
			continue
		}
		if stage.Started != 0 {
			stage.Status = enum.CIStatusKilled
		} else {
			stage.Status = enum.CIStatusSkipped
			stage.Started = now
		}
		stage.Stopped = now
		err := s.stageStore.Update(ctx, stage)
		if err != nil {
			log.Debug().Err(err).
				Int64("stage.number", stage.Number).
				Msg("canceler: cannot update stage status")
		}

		// update the status of all steps to indicate they
		// were killed or skipped.
		for _, step := range stage.Steps {
			if step.Status.IsDone() {
				continue
			}
			if step.Started != 0 {
				step.Status = enum.CIStatusKilled
			} else {
				step.Status = enum.CIStatusSkipped
				step.Started = now
			}
			step.Stopped = now
			step.ExitCode = 130
			err := s.stepStore.Update(ctx, step)
			if err != nil {
				log.Debug().Err(err).
					Int64("stage.number", stage.Number).
					Int64("step.number", step.Number).
					Msg("canceler: cannot update step status")
			}
		}
	}

	execution.Stages = stages
	log.Info().Msg("canceler: successfully cancelled build")

	s.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypeExecutionCanceled, execution)

	return nil
}
