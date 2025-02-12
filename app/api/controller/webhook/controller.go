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

package webhook

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	authorizer     authz.Authorizer
	spaceFinder    refcache.SpaceFinder
	repoFinder     refcache.RepoFinder
	webhookService *webhook.Service
	encrypter      encrypt.Encrypter
	preprocessor   Preprocessor
}

func NewController(
	authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	webhookService *webhook.Service,
	encrypter encrypt.Encrypter,
	preprocessor Preprocessor,
) *Controller {
	return &Controller{
		authorizer:     authorizer,
		spaceFinder:    spaceFinder,
		repoFinder:     repoFinder,
		webhookService: webhookService,
		encrypter:      encrypter,
		preprocessor:   preprocessor,
	}
}

//nolint:unparam
func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	allowedRepoStates ...enum.RepoState,
) (*types.RepositoryCore, error) {
	if repoRef == "" {
		return nil, errors.InvalidArgument("A valid repository reference must be provided.")
	}

	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, reqPermission, allowedRepoStates...); err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("failed to verify authorization: %w", err)
	}

	return repo, nil
}

func (c *Controller) getSpaceCheckAccess(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	permission enum.Permission,
) (*types.SpaceCore, error) {
	return space.GetSpaceCheckAuth(ctx, c.spaceFinder, c.authorizer, session, spaceRef, permission)
}
