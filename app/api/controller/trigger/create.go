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
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// TODO: Add more as needed.
type CreateInput struct {
	Description string `json:"description"`
	// TODO [CODE-1363]: remove after identifier migration.
	UID        string               `json:"uid" deprecated:"true"`
	Identifier string               `json:"identifier"`
	Secret     string               `json:"secret"`
	Disabled   bool                 `json:"disabled"`
	Actions    []enum.TriggerAction `json:"actions"`
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	in *CreateInput,
) (*types.Trigger, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	repo, err := c.getRepoCheckPipelineAccess(ctx, session, repoRef, pipelineIdentifier, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, err
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	now := time.Now().UnixMilli()
	trigger := &types.Trigger{
		Description: in.Description,
		Disabled:    in.Disabled,
		Secret:      in.Secret,
		CreatedBy:   session.Principal.ID,
		RepoID:      repo.ID,
		Actions:     deduplicateActions(in.Actions),
		Identifier:  in.Identifier,
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

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := check.Description(in.Description); err != nil {
		return err
	}
	if err := checkSecret(in.Secret); err != nil {
		return err
	}
	if err := checkActions(in.Actions); err != nil {
		return err
	}
	if err := check.Identifier(in.Identifier); err != nil { //nolint:revive
		return err
	}

	return nil
}
