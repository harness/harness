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

package lfs

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/remoteauth"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	lfsObjectsPathFormat = "lfs/%s"
)

type Controller struct {
	authorizer     authz.Authorizer
	repoFinder     refcache.RepoFinder
	principalStore store.PrincipalStore
	lfsStore       store.LFSObjectStore
	blobStore      blob.Store
	remoteAuth     remoteauth.Service
	urlProvider    url.Provider
}

func NewController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	principalStore store.PrincipalStore,
	lfsStore store.LFSObjectStore,
	blobStore blob.Store,
	remoteAuth remoteauth.Service,
	urlProvider url.Provider,
) *Controller {
	return &Controller{
		authorizer:     authorizer,
		repoFinder:     repoFinder,
		principalStore: principalStore,
		lfsStore:       lfsStore,
		blobStore:      blobStore,
		remoteAuth:     remoteAuth,
		urlProvider:    urlProvider,
	}
}

func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	allowedRepoStates ...enum.RepoState,
) (*types.RepositoryCore, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, reqPermission, allowedRepoStates...); err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}

func getLFSObjectPath(oid string) string {
	return fmt.Sprintf(lfsObjectsPathFormat, oid)
}
