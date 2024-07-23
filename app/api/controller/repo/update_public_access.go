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
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type UpdatePublicAccessInput struct {
	IsPublic bool `json:"is_public"`
}

func (c *Controller) UpdatePublicAccess(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *UpdatePublicAccessInput,
) (*RepositoryOutput, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	parentPath, _, err := paths.DisectLeaf(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to disect path %q: %w", repo.Path, err)
	}
	isPublicAccessSupported, err := c.publicAccess.IsPublicAccessSupported(ctx, parentPath)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to check if public access is supported for parent space %q: %w",
			parentPath,
			err,
		)
	}
	if in.IsPublic && !isPublicAccessSupported {
		return nil, errPublicRepoCreationDisabled
	}

	isPublic, err := c.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check current public access status: %w", err)
	}

	// no op
	if isPublic == in.IsPublic {
		return GetRepoOutputWithAccess(ctx, isPublic, repo), nil
	}

	if err = c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repo.Path, in.IsPublic); err != nil {
		return nil, fmt.Errorf("failed to update repo public access: %w", err)
	}

	// backfill GitURL
	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
		audit.ActionUpdated,
		paths.Parent(repo.Path),
		audit.WithOldObject(audit.RepositoryObject{
			Repository: *repo,
			IsPublic:   isPublic,
		}),
		audit.WithNewObject(audit.RepositoryObject{
			Repository: *repo,
			IsPublic:   in.IsPublic,
		}),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for update repository operation: %s", err)
	}

	return GetRepoOutputWithAccess(ctx, in.IsPublic, repo), nil
}
