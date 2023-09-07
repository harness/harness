// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package manager

import (
	"context"
	"errors"
	"time"

	"github.com/harness/gitness/internal/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type setup struct {
	Executions store.ExecutionStore
	Repos      store.RepoStore
	Steps      store.StepStore
	Stages     store.StageStore
	Users      store.PrincipalStore
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

	_, err = s.Repos.Find(noContext, execution.RepoID)
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
			Str("stage.status", stage.Status).
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
				Str("stage.status", stage.Status).
				Str("step.name", step.Name).
				Int64("step.id", step.ID).
				Msg("manager: cannot persist the step")
			return err
		}
	}

	_, err = s.updateExecution(ctx, execution)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot update the execution")
		return err
	}

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
	err := s.Executions.Update(noContext, execution)
	if errors.Is(err, gitness_store.ErrVersionConflict) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
