// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// TODO: Add more as needed.
type CreateInput struct {
	Description string               `json:"description"`
	UID         string               `json:"uid"`
	Secret      string               `json:"secret"`
	Enabled     bool                 `json:"enabled"`
	Actions     []enum.TriggerAction `json:"actions"`
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	in *CreateInput,
) (*types.Trigger, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	// Trigger permissions are associated with pipeline permissions. If a user has permissions
	// to edit the pipeline, they will have permissions to create a trigger as well.
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	err = c.checkCreateInput(in)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	now := time.Now().UnixMilli()
	trigger := &types.Trigger{
		Description: in.Description,
		Enabled:     in.Enabled,
		Secret:      in.Secret,
		CreatedBy:   session.Principal.ID,
		RepoID:      repo.ID,
		Actions:     deduplicateActions(in.Actions),
		UID:         in.UID,
		PipelineID:  pipeline.ID,
		Created:     now,
		Updated:     now,
		Version:     0,
	}
	err = c.triggerStore.Create(ctx, trigger)
	if err != nil {
		return nil, fmt.Errorf("trigger creation failed: %w", err)
	}

	return trigger, nil
}

func (c *Controller) checkCreateInput(in *CreateInput) error {
	if err := check.Description(in.Description); err != nil {
		return err
	}
	if err := checkSecret(in.Secret); err != nil {
		return err
	}
	if err := checkActions(in.Actions); err != nil {
		return err
	}
	if err := c.uidCheck(in.UID, false); err != nil {
		return err
	}

	return nil
}
