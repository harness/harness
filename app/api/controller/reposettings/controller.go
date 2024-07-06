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

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	authorizer   authz.Authorizer
	repoStore    store.RepoStore
	settings     *settings.Service
	auditService audit.Service
}

func NewController(
	authorizer authz.Authorizer,
	repoStore store.RepoStore,
	settings *settings.Service,
	auditService audit.Service,
) *Controller {
	return &Controller{
		authorizer:   authorizer,
		repoStore:    repoStore,
		settings:     settings,
		auditService: auditService,
	}
}

// getRepoCheckAccess fetches an active repo (not one that is currently being imported)
// and checks if the current user has permission to access it.
func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
) (*types.Repository, error) {
	return repo.GetRepoCheckAccess(
		ctx,
		c.repoStore,
		c.authorizer,
		session,
		repoRef,
		reqPermission,
		repo.ActiveRepoStates,
	)
}
