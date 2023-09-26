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

package trigger

import (
	"context"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
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
	Disabled    *bool                `json:"disabled"` // can be nil, so keeping it a pointer
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
			if in.UID != nil {
				original.UID = *in.UID
			}
			if in.Description != nil {
				original.Description = *in.Description
			}
			if in.Actions != nil {
				original.Actions = deduplicateActions(in.Actions)
			}
			if in.Secret != nil {
				original.Secret = *in.Secret
			}
			if in.Disabled != nil {
				original.Disabled = *in.Disabled
			}

			return nil
		})
}

func (c *Controller) checkUpdateInput(in *UpdateInput) error {
	if in.UID != nil {
		if err := c.uidCheck(*in.UID, false); err != nil {
			return err
		}
	}

	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
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

	return nil
}
