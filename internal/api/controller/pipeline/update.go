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

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type UpdateInput struct {
	UID         *string `json:"uid"`
	Description *string `json:"description"`
	Disabled    *bool   `json:"disabled"`
	ConfigPath  *string `json:"config_path"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	uid string,
	in *UpdateInput,
) (*types.Pipeline, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, uid, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	return c.pipelineStore.UpdateOptLock(ctx, pipeline, func(pipeline *types.Pipeline) error {
		if in.UID != nil {
			pipeline.UID = *in.UID
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
}

func (c *Controller) sanitizeUpdatenput(in *UpdateInput) error {
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

	if in.ConfigPath != nil {
		if *in.ConfigPath == "" {
			return errPipelineRequiresConfigPath
		}
	}

	return nil
}
