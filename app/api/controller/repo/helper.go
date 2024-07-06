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
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/slices"
)

var ActiveRepoStates = []enum.RepoState{enum.RepoStateActive}

// GetRepo fetches an repository.
func GetRepo(
	ctx context.Context,
	repoStore store.RepoStore,
	repoRef string,
	allowedStates []enum.RepoState,
) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if len(allowedStates) > 0 && !slices.Contains(allowedStates, repo.State) {
		return nil, usererror.BadRequest("Repository is not ready to use.")
	}

	return repo, nil
}

// GetRepoCheckAccess fetches an active repo (not one that is currently being imported)
// and checks if the current user has permission to access it.
func GetRepoCheckAccess(
	ctx context.Context,
	repoStore store.RepoStore,
	authorizer authz.Authorizer,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	allowedStates []enum.RepoState,
) (*types.Repository, error) {
	repo, err := GetRepo(ctx, repoStore, repoRef, allowedStates)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}

func GetRepoOutput(
	ctx context.Context,
	publicAccess publicaccess.Service,
	repo *types.Repository,
) (*RepositoryOutput, error) {
	isPublic, err := publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if repo is public: %w", err)
	}

	return &RepositoryOutput{
		Repository: *repo,
		IsPublic:   isPublic,
		Importing:  repo.State != enum.RepoStateActive,
	}, nil
}

func GetRepoOutputWithAccess(
	_ context.Context,
	isPublic bool,
	repo *types.Repository,
) *RepositoryOutput {
	return &RepositoryOutput{
		Repository: *repo,
		IsPublic:   isPublic,
		Importing:  repo.State != enum.RepoStateActive,
	}
}
