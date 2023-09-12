// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a trigger.
type UpdateInput struct {
	Description *string              `json:"description"`
	UID         *string              `json:"uid"`
	Actions     []enum.TriggerAction `json:"actions"`
	Secret      *string              `json:"secret"`
	Enabled     *bool                `json:"enabled"` // can be nil, so keeping it a pointer
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	triggerUID string,
	in *UpdateInput) (*types.Trigger, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	// Trigger permissions are associated with pipeline permissions. If a user has permissions
	// to edit the pipeline, they will have permissions to edit the trigger as well.
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	err = c.checkUpdateInput(in)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	trigger, err := c.triggerStore.FindByUID(ctx, pipeline.ID, triggerUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find trigger: %w", err)
	}

	return c.triggerStore.UpdateOptLock(ctx,
		trigger, func(original *types.Trigger) error {
			// update values only if provided
			if in.Description != nil {
				original.Description = *in.Description
			}
			if in.UID != nil {
				original.UID = *in.UID
			}
			if in.Actions != nil {
				original.Actions = deduplicateActions(in.Actions)
			}
			if in.Secret != nil {
				original.Secret = *in.Secret
			}
			if in.Enabled != nil {
				original.Enabled = *in.Enabled
			}

			return nil
		})
}

func (c *Controller) checkUpdateInput(in *UpdateInput) error {
	if in.Description != nil {
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}
	if in.Secret != nil {
		if err := checkSecret(*in.Secret); err != nil {
			return err
		}
	}
	if in.Actions != nil {
		if err := checkActions(in.Actions); err != nil {
			return err
		}
	}

	if in.UID != nil {
		if err := c.uidCheck(*in.UID, false); err != nil {
			return err
		}
	}

	return nil
}
