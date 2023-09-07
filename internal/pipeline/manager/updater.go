// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package manager

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type updater struct {
	Executions store.ExecutionStore
	Repos      store.RepoStore
	Steps      store.StepStore
	Stages     store.StageStore
}

func (u *updater) do(ctx context.Context, step *types.Step) error {
	log := log.With().
		Str("step.name", step.Name).
		Str("step.status", step.Status).
		Int64("step.id", step.ID).
		Logger()

	if len(step.Error) > 500 {
		step.Error = step.Error[:500]
	}
	err := u.Steps.Update(noContext, step)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot update step")
		return err
	}

	return nil
}
