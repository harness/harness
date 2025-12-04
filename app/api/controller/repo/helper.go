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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/slices"
)

var importingStates = []enum.RepoState{
	enum.RepoStateGitImport,
	enum.RepoStateMigrateDataImport,
	enum.RepoStateMigrateGitPush,
}

// GetRepo fetches a repository.
func GetRepo(
	ctx context.Context,
	repoFinder refcache.RepoFinder,
	repoRef string,
) (*types.RepositoryCore, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	return repo, nil
}

// GetRepoCheckAccess fetches a repo with option to enforce repo state check
// and checks if the current user has permission to access it.
func GetRepoCheckAccess(
	ctx context.Context,
	repoFinder refcache.RepoFinder,
	authorizer authz.Authorizer,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	allowLinked bool,
	allowedRepoStates ...enum.RepoState,
) (*types.RepositoryCore, error) {
	repo, err := GetRepo(ctx, repoFinder, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if !allowLinked && repo.Type == enum.RepoTypeLinked && reqPermission != enum.PermissionRepoView {
		return nil, errors.Forbidden("Changes are not allowed to a linked repository.")
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, reqPermission, allowedRepoStates...); err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}

func GetSpaceCheckAuthRepoCreation(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	authorizer authz.Authorizer,
	session *auth.Session,
	parentRef string,
) (*types.SpaceCore, error) {
	space, err := spaceFinder.FindByRef(ctx, parentRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}

	// create is a special case - check permission without specific resource
	err = apiauth.CheckSpaceScope(
		ctx,
		authorizer,
		session,
		space,
		enum.ResourceTypeRepo,
		enum.PermissionRepoCreate,
	)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	return space, nil
}

func GetRepoOutput(
	ctx context.Context,
	publicAccess publicaccess.Service,
	repoFinder refcache.RepoFinder,
	repo *types.Repository,
) (*RepositoryOutput, error) {
	isPublic, err := publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if repo is public: %w", err)
	}

	var upstreamRepo *types.RepositoryCore
	if repo.ForkID != 0 {
		upstreamRepo, err = repoFinder.FindByID(ctx, repo.ForkID)
		if err != nil {
			return nil, fmt.Errorf("failed to find repo fork %d: %w", repo.ForkID, err)
		}
	}

	return &RepositoryOutput{
		Repository: *repo,
		IsPublic:   isPublic,
		Importing:  slices.Contains(importingStates, repo.State),
		Archived:   repo.State == enum.RepoStateArchived,
		Upstream:   upstreamRepo,
	}, nil
}

func GetRepoOutputWithAccess(
	ctx context.Context,
	repoFinder refcache.RepoFinder,
	isPublic bool,
	repo *types.Repository,
) (*RepositoryOutput, error) {
	var upstreamRepo *types.RepositoryCore
	if repo.ForkID != 0 {
		var err error

		upstreamRepo, err = repoFinder.FindByID(ctx, repo.ForkID)
		if err != nil {
			return nil, fmt.Errorf("failed to find repo fork %d: %w", repo.ForkID, err)
		}
	}

	return &RepositoryOutput{
		Repository: *repo,
		IsPublic:   isPublic,
		Importing:  slices.Contains(importingStates, repo.State),
		Archived:   repo.State == enum.RepoStateArchived,
		Upstream:   upstreamRepo,
	}, nil
}

// GetRepoCheckServiceAccountAccess fetches a repo with option to enforce repo state check
// and checks if the current user has permission to access service accounts within repo.
func GetRepoCheckServiceAccountAccess(
	ctx context.Context,
	session *auth.Session,
	authorizer authz.Authorizer,
	repoRef string,
	reqPermission enum.Permission,
	repoFinder refcache.RepoFinder,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	allowedRepoStates ...enum.RepoState,
) (*types.RepositoryCore, error) {
	repo, err := GetRepo(ctx, repoFinder, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, reqPermission, allowedRepoStates...); err != nil {
		return nil, err
	}

	if err := apiauth.CheckServiceAccount(ctx, authorizer, session, spaceStore, repoStore,
		enum.ParentResourceTypeRepo, repo.ID, "", reqPermission,
	); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}
