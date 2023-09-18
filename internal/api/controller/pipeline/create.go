// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var (
	// errPipelineRequiresParent is returned if the user tries to create a pipeline without a parent space.
	errPipelineRequiresParent = usererror.BadRequest(
		"Parent space required - standalone pipelines are not supported.")

	// errPipelineRequiresConfigPath is returned if the user tries to create a pipeline with an empty config path.
	errPipelineRequiresConfigPath = usererror.BadRequest(
		"Pipeline requires a config path.")
)

type CreateInput struct {
	Description   string `json:"description"`
	UID           string `json:"uid"`
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
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, "", enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	var pipeline *types.Pipeline
	now := time.Now().UnixMilli()
	pipeline = &types.Pipeline{
		Description:   in.Description,
		RepoID:        repo.ID,
		UID:           in.UID,
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
		UID:         "default",
		Actions: []enum.TriggerAction{enum.TriggerActionPullReqCreated,
			enum.TriggerActionPullReqReopened, enum.TriggerActionPullReqBranchUpdated},
		Disabled: false,
		Version:  0,
	}
	err = c.triggerStore.Create(ctx, trigger)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to create auto trigger on pipeline creation")
	}

	return pipeline, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if err := c.uidCheck(in.UID, false); err != nil {
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
