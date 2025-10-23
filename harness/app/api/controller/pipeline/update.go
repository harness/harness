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

	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type UpdateInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID         *string `json:"uid" deprecated:"true"`
	Identifier  *string `json:"identifier"`
	Description *string `json:"description"`
	Disabled    *bool   `json:"disabled"`
	ConfigPath  *string `json:"config_path"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	identifier string,
	in *UpdateInput,
) (*types.Pipeline, error) {
	repo, err := c.getRepoCheckPipelineAccess(ctx, session, repoRef, identifier, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, err
	}

	if err = c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	updated, err := c.pipelineStore.UpdateOptLock(ctx, pipeline, func(pipeline *types.Pipeline) error {
		if in.Identifier != nil {
			pipeline.Identifier = *in.Identifier
		}
		if in.Description != nil {
			pipeline.Description = *in.Description
		}
		if in.ConfigPath != nil {
			pipeline.ConfigPath = *in.ConfigPath
		}
		if in.Disabled != nil {
			pipeline.Disabled = *in.Disabled
		}

		return nil
	})

	// send pipeline update event
	c.reporter.Updated(ctx, &events.UpdatedPayload{PipelineID: pipeline.ID, RepoID: pipeline.RepoID})

	return updated, err
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := check.Identifier(*in.Identifier); err != nil {
			return err
		}
	}

	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	if in.ConfigPath != nil {
		if *in.ConfigPath == "" {
			return errPipelineRequiresConfigPath
		}
	}

	return nil
}
