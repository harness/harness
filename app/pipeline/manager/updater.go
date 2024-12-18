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

	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type updater struct {
	Executions  store.ExecutionStore
	Repos       store.RepoStore
	SSEStreamer sse.Streamer
	Steps       store.StepStore
	Stages      store.StageStore
}

func (u *updater) do(ctx context.Context, step *types.Step) error {
	log := log.Ctx(ctx).With().
		Str("step.name", step.Name).
		Str("step.status", string(step.Status)).
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

	stage, err := u.Stages.Find(noContext, step.StageID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find stage")
		return nil
	}

	execution, err := u.Executions.Find(noContext, stage.ExecutionID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find execution")
		return nil
	}

	repo, err := u.Repos.Find(noContext, execution.RepoID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find repo")
		return nil
	}

	stages, err := u.Stages.ListWithSteps(noContext, stage.ExecutionID)
	if err != nil {
		log.Error().Err(err).Msg("manager: cannot find stages")
		return nil
	}
	execution.Stages = stages

	u.SSEStreamer.Publish(noContext, repo.ParentID, enum.SSETypeExecutionUpdated, execution)

	return nil
}
