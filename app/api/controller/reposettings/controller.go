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

package reposettings

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	authorizer   authz.Authorizer
	repoFinder   refcache.RepoFinder
	spaceFinder  refcache.SpaceFinder
	settings     *settings.Service
	auditService audit.Service
}

func NewController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	spaceFinder refcache.SpaceFinder,
	settings *settings.Service,
	auditService audit.Service,
) *Controller {
	return &Controller{
		authorizer:   authorizer,
		repoFinder:   repoFinder,
		spaceFinder:  spaceFinder,
		settings:     settings,
		auditService: auditService,
	}
}

// getRepoCheckAccess fetches a repo, checks if operation is allowed given the repo state
// and checks if the current user has permission to access it.
func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	allowedRepoStates ...enum.RepoState,
) (*types.RepositoryCore, error) {
	repo, err := repo.GetRepo(ctx, c.repoFinder, repoRef)
	if err != nil {
		return nil, err
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, reqPermission, allowedRepoStates...); err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}

func (c *Controller) getSpaceCheckAccess(
	ctx context.Context,
	session *auth.Session,
	parentRef string,
	reqPermission enum.Permission,
) (*types.SpaceCore, error) {
	space, err := c.spaceFinder.FindByRef(ctx, parentRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}

	err = apiauth.CheckSpaceScope(
		ctx,
		c.authorizer,
		session,
		space,
		enum.ResourceTypeSpace,
		reqPermission,
	)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	return space, nil
}
