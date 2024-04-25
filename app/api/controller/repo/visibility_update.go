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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type VisibilityInput struct {
	EnablePublic bool `json:"enable_public"`
}

type VisibilityOutput struct {
	IsPublic bool `json:"is_public"`
}

func (c *Controller) VisibilityUpdate(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *VisibilityInput,
) (*VisibilityOutput, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	if err = c.sanitizeVisibilityInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, name, err := paths.DisectLeaf(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("Failed to disect path '%s': %w", repo.Path, err)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type:       enum.ResourceTypeRepo,
		Identifier: name,
	}

	if err = c.publicAccess.Set(ctx, scope, resource, in.EnablePublic); err != nil {
		return nil, fmt.Errorf("failed to set public access: %w", err)
	}

	return &VisibilityOutput{
		in.EnablePublic,
	}, nil

}

func (c *Controller) sanitizeVisibilityInput(in *VisibilityInput) error {
	if in.EnablePublic && !c.publicResourceCreationEnabled {
		return errPublicRepoCreationDisabled
	}

	return nil
}
