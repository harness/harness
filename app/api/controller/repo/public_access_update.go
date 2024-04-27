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
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

type PublicAccessUpdateInput struct {
	EnablePublic bool `json:"enable_public"`
}

type PublicAccessUpdateOutput struct {
	IsPublic bool `json:"is_public"`
}

func (c *Controller) PublicAccessUpdate(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *PublicAccessUpdateInput,
) (*PublicAccessUpdateOutput, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	repoClone := repo.Clone()

	if err = c.sanitizeVisibilityInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, name, err := paths.DisectLeaf(repo.Repository.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to disect path '%s': %w", repo.Repository.Path, err)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type:       enum.ResourceTypeRepo,
		Identifier: name,
	}

	if err = c.publicAccess.Set(ctx, scope, resource, in.EnablePublic); err != nil {
		return nil, fmt.Errorf("failed to set public access: %w", err)
	}

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repo.Repository.Identifier),
		audit.ActionUpdated,
		paths.Space(repo.Repository.Path),
		audit.WithOldObject(repoClone),
		audit.WithNewObject(&Repository{
			Repository: repo.Repository,
			IsPublic:   in.EnablePublic,
		}),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for update repository operation: %s", err)
	}

	return &PublicAccessUpdateOutput{
		in.EnablePublic,
	}, nil

}

func (c *Controller) sanitizeVisibilityInput(in *PublicAccessUpdateInput) error {
	if in.EnablePublic && !c.publicResourceCreationEnabled {
		return errPublicRepoCreationDisabled
	}

	return nil
}
