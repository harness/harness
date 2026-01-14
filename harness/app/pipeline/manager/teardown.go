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
	"strings"
	"time"

	events "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/app/pipeline/checks"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/livelog"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
)

type teardown struct {
	Executions  store.ExecutionStore
	Checks      store.CheckStore
	Pipelines   store.PipelineStore
	SSEStreamer sse.Streamer
	Logs        livelog.LogStream
	Scheduler   scheduler.Scheduler
	Repos       store.RepoStore
	Steps       store.StepStore
	Stages      store.StageStore
	Reporter    events.Reporter
}

//nolint:gocognit // refactor if needed.
func (t *teardown) do(ctx context.Context, stage *types.Stage) error {
	log := log.With().
		Int64("stage.id", stage.ID).
		Logger()
	log.Debug().Msg("manager: stage is complete. teardown")

	execution, err := t.Executions.Find(noContext, stage.ExecutionID) //nolint:contextcheck
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find the execution")
		return err
	}

	log = log.With().
		Int64("execution.number", execution.Number).
		Int64("execution.id", execution.ID).
		Int64("repo.id", execution.RepoID).
		Str("stage.status", string(stage.Status)).
		Logger()

	repo, err := t.Repos.Find(noContext, execution.RepoID) //nolint:contextcheck
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find the repository")
		return err
	}

	for _, step := range stage.Steps {
		if len(step.Error) > 500 {
			step.Error = step.Error[:500]
		}
		err := t.Steps.Update(noContext, step) //nolint:contextcheck
		if err != nil {
			log = log.With().
				Str("step.name", step.Name).
				Int64("step.id", step.ID).
				Err(err).
				Logger()

			log.Error().Msg("manager: cannot persist the step")
			return err
		}
	}

	if len(stage.Error) > 500 {
		stage.Error = stage.Error[:500]
	}

	err = t.Stages.Update(noContext, stage) //nolint:contextcheck
	if err != nil {
		log.Error().Err(err).
			Msg("manager: cannot update the stage")
		return err
	}

	for _, step := range stage.Steps {
		err = t.Logs.Delete(noContext, step.ID) //nolint:contextcheck
		if err != nil && !errors.Is(err, livelog.ErrStreamNotFound) {
			log.Warn().Err(err).Msgf("failed to delete log stream for step %d", step.ID)
		}
	}

	stages, err := t.Stages.ListWithSteps(noContext, execution.ID) //nolint:contextcheck
	if err != nil {
		log.Warn().Err(err).
			Msg("manager: cannot get stages")
		return err
	}

	err = t.cancelDownstream(ctx, stages)
	if err != nil {
		log.Error().Err(err).
			Msg("manager: cannot cancel downstream builds")
		return err
	}

	err = t.scheduleDownstream(ctx, stages)
	if err != nil {
		log.Error().Err(err).
			Msg("manager: cannot schedule downstream builds")
		return err
	}

	if !isexecutionComplete(stages) {
		log.Warn().Err(err).
			Msg("manager: execution pending completion of additional stages")
		return nil
	}

	log.Info().Msg("manager: execution is finished, teardown")

	execution.Status = enum.CIStatusSuccess
	execution.Finished = time.Now().UnixMilli()
	for _, sibling := range stages {
		if sibling.Status == enum.CIStatusKilled {
			execution.Status = enum.CIStatusKilled
			break
		}
		if sibling.Status == enum.CIStatusFailure {
			execution.Status = enum.CIStatusFailure
			break
		}
		if sibling.Status == enum.CIStatusError {
			execution.Status = enum.CIStatusError
			break
		}
	}
	if execution.Started == 0 {
		execution.Started = execution.Finished
	}

	err = t.Executions.Update(noContext, execution) //nolint:contextcheck
	if errors.Is(err, gitness_store.ErrVersionConflict) {
		log.Warn().Err(err).
			Msg("manager: execution updated by another goroutine")
		return nil
	}
	if err != nil {
		log.Warn().Err(err).
			Msg("manager: cannot update the execution")
		return err
	}

	execution.Stages = stages

	t.SSEStreamer.Publish(noContext, repo.ParentID, enum.SSETypeExecutionCompleted, execution) //nolint:contextcheck

	// send pipeline execution status
	t.reportExecutionCompleted(ctx, execution)

	pipeline, err := t.Pipelines.Find(ctx, execution.PipelineID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find pipeline")
		return err
	}
	// try to write to the checks store - if not, log an error and continue
	err = checks.Write(ctx, t.Checks, execution, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("manager: could not write to checks store")
	}

	return nil
}

// cancelDownstream is a helper function that tests for
// downstream stages and cancels them based on the overall
// pipeline state.
//
//nolint:gocognit // refactor if needed
func (t *teardown) cancelDownstream(
	ctx context.Context,
	stages []*types.Stage,
) error {
	failed := false
	for _, s := range stages {
		// check pipeline state
		if s.Status.IsFailed() {
			failed = true
		}
	}

	var errs error
	for _, s := range stages {
		if s.Status != enum.CIStatusWaitingOnDeps {
			continue
		}

		var skip bool
		if failed && !s.OnFailure {
			skip = true
		}
		if !failed && !s.OnSuccess {
			skip = true
		}
		if !skip {
			continue
		}

		if !areDepsComplete(s, stages) {
			continue
		}

		log := log.With().
			Int64("stage.id", s.ID).
			Bool("stage.on_success", s.OnSuccess).
			Bool("stage.on_failure", s.OnFailure).
			Bool("failed", failed).
			Str("stage.depends_on", strings.Join(s.DependsOn, ",")).
			Logger()

		log.Debug().Msg("manager: skipping step")

		s.Status = enum.CIStatusSkipped
		s.Started = time.Now().UnixMilli()
		s.Stopped = time.Now().UnixMilli()
		err := t.Stages.Update(noContext, s) //nolint:contextcheck
		if errors.Is(err, gitness_store.ErrVersionConflict) {
			rErr := t.resync(ctx, s)
			if rErr != nil {
				log.Warn().Err(rErr).Msg("failed to resync after version conflict")
			}
			continue
		}
		if err != nil {
			log.Error().Err(err).
				Msg("manager: cannot update stage status")
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

func isexecutionComplete(stages []*types.Stage) bool {
	for _, stage := range stages {
		if stage.Status == enum.CIStatusPending ||
			stage.Status == enum.CIStatusRunning ||
			stage.Status == enum.CIStatusWaitingOnDeps ||
			stage.Status == enum.CIStatusDeclined ||
			stage.Status == enum.CIStatusBlocked {
			return false
		}
	}
	return true
}

func areDepsComplete(stage *types.Stage, stages []*types.Stage) bool {
	deps := map[string]struct{}{}
	for _, dep := range stage.DependsOn {
		deps[dep] = struct{}{}
	}
	for _, sibling := range stages {
		if _, ok := deps[sibling.Name]; !ok {
			continue
		}
		if !sibling.Status.IsDone() {
			return false
		}
	}
	return true
}

// scheduleDownstream is a helper function that tests for
// downstream stages and schedules stages if all dependencies
// and execution requirements are met.
func (t *teardown) scheduleDownstream(
	ctx context.Context,
	stages []*types.Stage,
) error {
	var errs error
	for _, sibling := range stages {
		if sibling.Status != enum.CIStatusWaitingOnDeps {
			continue
		}

		if len(sibling.DependsOn) == 0 {
			continue
		}

		// PROBLEM: isDep only checks the direct parent
		// i think ....
		// if isDep(stage, sibling) == false {
		// 	continue
		// }
		if !areDepsComplete(sibling, stages) {
			continue
		}
		// if isLastDep(stage, sibling, stages) == false {
		// 	continue
		// }

		log := log.With().
			Int64("stage.id", sibling.ID).
			Str("stage.name", sibling.Name).
			Str("stage.depends_on", strings.Join(sibling.DependsOn, ",")).
			Logger()

		log.Debug().Msg("manager: schedule next stage")

		sibling.Status = enum.CIStatusPending
		err := t.Stages.Update(noContext, sibling) //nolint:contextcheck
		if errors.Is(err, gitness_store.ErrVersionConflict) {
			rErr := t.resync(ctx, sibling)
			if rErr != nil {
				log.Warn().Err(rErr).Msg("failed to resync after version conflict")
			}
			continue
		}
		if err != nil {
			log.Error().Err(err).
				Msg("manager: cannot update stage status")
			errs = multierror.Append(errs, err)
		}

		err = t.Scheduler.Schedule(noContext, sibling) //nolint:contextcheck
		if err != nil {
			log.Error().Err(err).
				Msg("manager: cannot schedule stage")
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// resync updates the stage from the database. Note that it does
// not update the Version field. This is by design. It prevents
// the current go routine from updating a stage that has been
// updated by another go routine.
func (t *teardown) resync(ctx context.Context, stage *types.Stage) error {
	updated, err := t.Stages.Find(ctx, stage.ID)
	if err != nil {
		return err
	}
	stage.Status = updated.Status
	stage.Error = updated.Error
	stage.ExitCode = updated.ExitCode
	stage.Machine = updated.Machine
	stage.Started = updated.Started
	stage.Stopped = updated.Stopped
	return nil
}

func (t *teardown) reportExecutionCompleted(ctx context.Context, execution *types.Execution) {
	t.Reporter.Executed(ctx, &events.ExecutedPayload{
		PipelineID:   execution.PipelineID,
		RepoID:       execution.RepoID,
		ExecutionNum: execution.Number,
		Status:       execution.Status,
	})
}
