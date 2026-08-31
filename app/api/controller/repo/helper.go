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
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

// transientStates are states with no restorable data behind them.
// Deleting a repository in one of these states purges it immediately
// instead of soft-deleting it into a (non-restorable) trash.
var transientStates = []enum.RepoState{
	enum.RepoStateGitImport,
	enum.RepoStateMigrateDataImport,
	enum.RepoStateMigrateGitPush,
	enum.RepoStateImportFailed,
}

// importingStates are the states that represent an in-progress import.
// They drive the `importing` output flag. A failed import is terminal, not
// in-progress, so it's deliberately excluded here (see RepoStateImportFailed).
var importingStates = []enum.RepoState{
	enum.RepoStateGitImport,
	enum.RepoStateMigrateDataImport,
	enum.RepoStateMigrateGitPush,
}

// EnsureIdentifierAvailable verifies that no repository with the provided identifier exists in the
// parent space, so that a new one can be created there.
//
// A repository whose import failed is kept in the DB solely to surface the failure to the user - it
// has no git data behind it and is a dead object. Such a repository is removed here, freeing up the
// identifier, so that the caller can retry the creation/import without a manual cleanup first.
//
// Any other existing repository is reported back as a conflict.
func (c *Controller) EnsureIdentifierAvailable(
	ctx context.Context,
	session *auth.Session,
	parentSpaceID int64,
	identifier string,
) error {
	repo, err := c.repoStore.FindActiveByUID(ctx, parentSpaceID, identifier)
	if errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check for an existing repository with the same identifier: %w", err)
	}

	if repo.State != enum.RepoStateImportFailed {
		return usererror.Conflict(fmt.Sprintf(
			"A repository with identifier %q already exists in this space.", identifier))
	}

	log.Ctx(ctx).Info().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Msg("removing repository with failed import to free up its identifier")

	if err := c.publicAccess.Delete(ctx, enum.PublicResourceTypeRepo, repo.Path); err != nil {
		return fmt.Errorf("failed to delete public access of repository with failed import: %w", err)
	}

	// there's no data behind the repository, so purge it right away rather than move it to the trash.
	if err := c.PurgeNoAuth(ctx, session, repo); err != nil {
		return fmt.Errorf("failed to purge repository with failed import: %w", err)
	}

	c.repoFinder.MarkChanged(ctx, repo.Core())

	return nil
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

	repoClone := *repo

	var upstreamRepo *types.RepositoryCore
	if repo.ForkID != 0 {
		upstreamRepo, err = repoFinder.FindByID(ctx, repo.ForkID)
		if errors.Is(err, gitnessstore.ErrResourceNotFound) {
			repoClone.ForkID = 0
		} else if err != nil {
			return nil, fmt.Errorf("failed to find repo fork %d: %w", repo.ForkID, err)
		}
	}

	return &RepositoryOutput{
		Repository:   repoClone,
		IsPublic:     isPublic,
		Importing:    slices.Contains(importingStates, repo.State),
		ImportFailed: repo.State == enum.RepoStateImportFailed,
		Archived:     repo.State == enum.RepoStateArchived,
		Upstream:     upstreamRepo,
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
		Repository:   *repo,
		IsPublic:     isPublic,
		Importing:    slices.Contains(importingStates, repo.State),
		ImportFailed: repo.State == enum.RepoStateImportFailed,
		Archived:     repo.State == enum.RepoStateArchived,
		Upstream:     upstreamRepo,
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
