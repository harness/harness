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

package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var (
	// errPipelineRequiresConfigPath is returned if the user tries to create a pipeline with an empty config path.
	errPipelineRequiresConfigPath = usererror.BadRequest(
		"Pipeline requires a config path.")
)

type CreateInput struct {
	Description string `json:"description"`
	// TODO [CODE-1363]: remove after identifier migration.
	UID           string `json:"uid" deprecated:"true"`
	Identifier    string `json:"identifier"`
	Disabled      bool   `json:"disabled"`
	DefaultBranch string `json:"default_branch"`
	ConfigPath    string `json:"config_path"`
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
) (*types.Pipeline, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	repo, err := c.getRepoCheckPipelineAccess(ctx, session, repoRef, "", enum.PermissionPipelineEdit)
	if err != nil {
		return nil, err
	}

	var pipeline *types.Pipeline
	now := time.Now().UnixMilli()
	pipeline = &types.Pipeline{
		Description:   in.Description,
		RepoID:        repo.ID,
		Identifier:    in.Identifier,
		Disabled:      in.Disabled,
		CreatedBy:     session.Principal.ID,
		Seq:           0,
		DefaultBranch: in.DefaultBranch,
		ConfigPath:    in.ConfigPath,
		Created:       now,
		Updated:       now,
		Version:       0,
	}
	err = c.pipelineStore.Create(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("pipeline creation failed: %w", err)
	}

	// Try to create a default trigger on pipeline creation.
	// Default trigger operations are set on pull request created, reopened or updated.
	// We log an error on failure but don't fail the op.
	trigger := &types.Trigger{
		Description: "auto-created trigger on pipeline creation",
		Created:     now,
		Updated:     now,
		PipelineID:  pipeline.ID,
		RepoID:      pipeline.RepoID,
		CreatedBy:   session.Principal.ID,
		Identifier:  "default",
		Actions: []enum.TriggerAction{enum.TriggerActionPullReqCreated,
			enum.TriggerActionPullReqReopened, enum.TriggerActionPullReqBranchUpdated},
		Disabled: false,
		Version:  0,
	}
	err = c.triggerStore.Create(ctx, trigger)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to create auto trigger on pipeline creation")
	}

	// send pipeline create event
	c.reporter.Created(ctx, &events.CreatedPayload{PipelineID: pipeline.ID, RepoID: pipeline.RepoID})

	return pipeline, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	if err := check.Description(in.Description); err != nil {
		return err
	}

	if in.DefaultBranch == "" {
		in.DefaultBranch = c.defaultBranch
	}

	if in.ConfigPath == "" {
		return errPipelineRequiresConfigPath
	}

	return nil
}
