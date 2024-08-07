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
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	authorizer      authz.Authorizer
	publicAccess    publicaccess.Service
	git             git.Interface
	urlProvider     url.Provider
	pullreqImporter *migrate.PullReq
	ruleImporter    *migrate.Rule
	webhookImporter *migrate.Webhook
	resourceLimiter limiter.ResourceLimiter
	auditService    audit.Service
	identifierCheck check.RepoIdentifier
	tx              dbtx.Transactor
	spaceStore      store.SpaceStore
	repoStore       store.RepoStore
}

func NewController(
	authorizer authz.Authorizer,
	publicAccess publicaccess.Service,
	git git.Interface,
	urlProvider url.Provider,
	pullreqImporter *migrate.PullReq,
	ruleImporter *migrate.Rule,
	webhookImporter *migrate.Webhook,
	resourceLimiter limiter.ResourceLimiter,
	auditService audit.Service,
	identifierCheck check.RepoIdentifier,
	tx dbtx.Transactor,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
) *Controller {
	return &Controller{
		authorizer:      authorizer,
		publicAccess:    publicAccess,
		git:             git,
		urlProvider:     urlProvider,
		pullreqImporter: pullreqImporter,
		ruleImporter:    ruleImporter,
		webhookImporter: webhookImporter,
		resourceLimiter: resourceLimiter,
		auditService:    auditService,
		identifierCheck: identifierCheck,
		tx:              tx,
		spaceStore:      spaceStore,
		repoStore:       repoStore,
	}
}

func (c *Controller) getRepoCheckAccess(ctx context.Context,
	session *auth.Session, repoRef string, reqPermission enum.Permission) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("failed to verify authorization: %w", err)
	}

	return repo, nil
}
