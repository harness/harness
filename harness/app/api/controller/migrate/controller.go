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
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
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
	labelImporter   *migrate.Label
	resourceLimiter limiter.ResourceLimiter
	auditService    audit.Service
	identifierCheck check.RepoIdentifier
	tx              dbtx.Transactor
	spaceStore      store.SpaceStore
	repoStore       store.RepoStore
	spaceFinder     refcache.SpaceFinder
	repoFinder      refcache.RepoFinder
	eventReporter   *repoevents.Reporter
}

func NewController(
	authorizer authz.Authorizer,
	publicAccess publicaccess.Service,
	git git.Interface,
	urlProvider url.Provider,
	pullreqImporter *migrate.PullReq,
	ruleImporter *migrate.Rule,
	webhookImporter *migrate.Webhook,
	labelImporter *migrate.Label,
	resourceLimiter limiter.ResourceLimiter,
	auditService audit.Service,
	identifierCheck check.RepoIdentifier,
	tx dbtx.Transactor,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	eventReporter *repoevents.Reporter,
) *Controller {
	return &Controller{
		authorizer:      authorizer,
		publicAccess:    publicAccess,
		git:             git,
		urlProvider:     urlProvider,
		pullreqImporter: pullreqImporter,
		ruleImporter:    ruleImporter,
		webhookImporter: webhookImporter,
		labelImporter:   labelImporter,
		resourceLimiter: resourceLimiter,
		auditService:    auditService,
		identifierCheck: identifierCheck,
		tx:              tx,
		spaceStore:      spaceStore,
		repoStore:       repoStore,
		spaceFinder:     spaceFinder,
		repoFinder:      repoFinder,
		eventReporter:   eventReporter,
	}
}

func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
) (*types.RepositoryCore, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	// repo state check happens per operation as it varies given the stage of the migration.

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("failed to verify authorization: %w", err)
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
