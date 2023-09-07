// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package manager

import (
	"context"
	"time"

	"github.com/harness/gitness/internal/pipeline/scheduler"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/livelog"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

type teardown struct {
	Executions store.ExecutionStore
	Logs       livelog.LogStream
	Scheduler  scheduler.Scheduler
	Repos      store.RepoStore
	Steps      store.StepStore
	Stages     store.StageStore
}

func (t *teardown) do(ctx context.Context, stage *types.Stage) error {
	log := log.With().
		Int64("stage.id", stage.ID).
		Logger()
	log.Debug().Msg("manager: stage is complete. teardown")

	execution, err := t.Executions.Find(noContext, stage.ExecutionID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find the execution")
		return err
	}

	log = log.With().
		Int64("execution.number", execution.Number).
		Int64("execution.id", execution.ID).
		Int64("repo.id", execution.RepoID).
		Str("stage.status", stage.Status).
		Logger()

	_, err = t.Repos.Find(noContext, execution.RepoID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find the repository")
		return err
	}

	for _, step := range stage.Steps {
		if len(step.Error) > 500 {
			step.Error = step.Error[:500]
		}
		err := t.Steps.Update(noContext, step)
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

	err = t.Stages.Update(noContext, stage)
	if err != nil {
		log.Error().Err(err).
			Msg("manager: cannot update the stage")
		return err
	}

	for _, step := range stage.Steps {
		t.Logs.Delete(noContext, step.ID)
	}

	stages, err := t.Stages.ListWithSteps(noContext, execution.ID)
	if err != nil {
		log.Warn().Err(err).
			Msg("manager: cannot get stages")
		return err
	}

	if isexecutionComplete(stages) == false {
		log.Warn().Err(err).
			Msg("manager: execution pending completion of additional stages")
		return nil
	}

	log.Info().Msg("manager: execution is finished, teardown")

	execution.Status = enum.StatusPassing
	execution.Finished = time.Now().Unix()
	for _, sibling := range stages {
		if sibling.Status == enum.StatusKilled {
			execution.Status = enum.StatusKilled
			break
		}
		if sibling.Status == enum.StatusFailing {
			execution.Status = enum.StatusFailing
			break
		}
		if sibling.Status == enum.StatusError {
			execution.Status = enum.StatusError
			break
		}
	}
	if execution.Started == 0 {
		execution.Started = execution.Finished
	}

	err = t.Executions.Update(noContext, execution)
	if err == gitness_store.ErrVersionConflict {
		log.Warn().Err(err).
			Msg("manager: execution updated by another goroutine")
		return nil
	}
	if err != nil {
		log.Warn().Err(err).
			Msg("manager: cannot update the execution")
		return err
	}

	return nil
}

func isexecutionComplete(stages []*types.Stage) bool {
	for _, stage := range stages {
		switch stage.Status {
		case enum.StatusPending,
			enum.StatusRunning,
			enum.StatusWaiting,
			enum.StatusDeclined,
			enum.StatusBlocked:
			return false
		}
	}
	return true
}
