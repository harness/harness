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

package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	repoCtrl "github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CreateRepoInput struct {
	ParentRef     string `json:"parent_ref"`
	Identifier    string `json:"identifier"`
	DefaultBranch string `json:"default_branch"`
	IsPublic      bool   `json:"is_public"`
}

func (c *Controller) CreateRepo(
	ctx context.Context,
	session *auth.Session,
	in *CreateRepoInput,
) (*repoCtrl.RepositoryOutput, error) {
	if err := c.sanitizeCreateRepoInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.spaceCheckAuth(ctx, session, in.ParentRef)
	if err != nil {
		return nil, fmt.Errorf("failed to check auth in parent '%s': %w", in.ParentRef, err)
	}

	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		c.urlProvider.GetInternalAPIURL(ctx),
		0,
		session.Principal.ID,
		true,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	actor := &git.Identity{
		Name:  session.Principal.DisplayName,
		Email: session.Principal.Email,
	}
	committer := bootstrap.NewSystemServiceSession().Principal
	now := time.Now()

	gitResp, err := c.git.CreateRepository(ctx, &git.CreateRepositoryParams{
		Actor:         *actor,
		EnvVars:       envVars,
		DefaultBranch: in.DefaultBranch,
		Files:         nil,
		Author:        actor,
		AuthorDate:    &now,
		Committer: &git.Identity{
			Name:  committer.DisplayName,
			Email: committer.Email,
		},
		CommitterDate: &now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create git repository: %w", err)
	}

	var repo *types.Repository
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(ctx, parentSpace.ID, 1); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		// lock the space for update during repo creation to prevent racing conditions with space soft delete.
		parentSpace, err = c.spaceStore.FindForUpdate(ctx, parentSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to find the parent space: %w", err)
		}

		repo = &types.Repository{
			Version:       0,
			ParentID:      parentSpace.ID,
			Identifier:    in.Identifier,
			GitUID:        gitResp.UID,
			CreatedBy:     session.Principal.ID,
			Created:       now.UnixMilli(),
			Updated:       now.UnixMilli(),
			DefaultBranch: in.DefaultBranch,
			IsEmpty:       true,
			State:         enum.RepoStateMigrateGitPush,
		}

		return c.repoStore.Create(ctx, repo)
	}, sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		// TODO: best effort cleanup
		return nil, fmt.Errorf("failed to create a repo on db: %w", err)
	}

	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	isPublicAccessSupported, err := c.publicAccess.IsPublicAccessSupported(ctx, parentSpace.Path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to check if public access is supported for parent space %s: %w",
			parentSpace.Path,
			err,
		)
	}

	isRepoPublic := in.IsPublic
	if !isPublicAccessSupported {
		log.Debug().Msgf("public access is not supported, create migrating repo %s as private instead", repo.Identifier)
		isRepoPublic = false
	}
	err = c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repo.Path, isRepoPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to set repo access mode: %w", err)
	}

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
		audit.ActionCreated,
		paths.Parent(repo.Path),
		audit.WithNewObject(audit.RepositoryObject{
			Repository: *repo,
			IsPublic:   isRepoPublic,
		}),
		audit.WithData("created by", "migrator"),
	)
	if err != nil {
		log.Warn().Msgf("failed to insert audit log for import repository operation: %s", err)
	}

	return &repoCtrl.RepositoryOutput{
		Repository: *repo,
		IsPublic:   isRepoPublic,
	}, nil
}

func (c *Controller) spaceCheckAuth(
	ctx context.Context,
	session *auth.Session,
	parentRef string,
) (*types.Space, error) {
	space, err := c.spaceStore.FindByRef(ctx, parentRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}

	// create is a special case - check permission without specific resource
	scope := &types.Scope{SpacePath: space.Path}
	resource := &types.Resource{
		Type:       enum.ResourceTypeRepo,
		Identifier: "",
	}

	err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	return space, nil
}

func (c *Controller) sanitizeCreateRepoInput(in *CreateRepoInput) error {
	if err := repoCtrl.ValidateParentRef(in.ParentRef); err != nil {
		return err
	}

	if err := c.identifierCheck(in.Identifier); err != nil {
		return err
	}

	return nil
}
