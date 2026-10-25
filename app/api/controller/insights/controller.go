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

// Package insights provides the controller of the repository scoped Insights admin API.
package insights

import (
	"context"
	"fmt"
	"strconv"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/insights/ingest"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

// Controller handles the Insights admin API.
type Controller struct {
	authorizer  authz.Authorizer
	repoFinder  refcache.RepoFinder
	urlProvider url.Provider
	service     *insights.Service
	audit       audit.Service
}

// NewController returns a new Controller.
func NewController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	urlProvider url.Provider,
	service *insights.Service,
	auditService audit.Service,
) *Controller {
	return &Controller{
		authorizer:  authorizer,
		repoFinder:  repoFinder,
		urlProvider: urlProvider,
		service:     service,
		audit:       auditService,
	}
}

// ProvideController provides the Insights admin controller.
func ProvideController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	urlProvider url.Provider,
	service *insights.Service,
	auditService audit.Service,
) *Controller {
	return NewController(authorizer, repoFinder, urlProvider, service, auditService)
}

func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
) (*types.RepositoryCore, error) {
	r, err := repo.GetRepo(ctx, c.repoFinder, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, r, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return r, nil
}

// dsn returns the DSN SDKs of the project have to use.
func (c *Controller) dsn(ctx context.Context, p *types.InsightsProject) string {
	return ingest.FormatDSN(
		c.urlProvider.GetAPIProto(ctx), p.PublicKey, c.urlProvider.GetAPIHostname(ctx),
		strconv.FormatInt(p.ID, 10))
}

func (c *Controller) withDSN(ctx context.Context, p *types.InsightsProject) *types.InsightsProjectWithDSN {
	return &types.InsightsProjectWithDSN{InsightsProject: *p, DSN: c.dsn(ctx, p)}
}

// CreateProject creates an Insights project in the repository.
func (c *Controller) CreateProject(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *types.InsightsProjectCreateInput,
) (*types.InsightsProjectWithDSN, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	p, err := c.service.CreateProject(ctx, r.ID, session.Principal.ID, in)
	if err != nil {
		return nil, err
	}

	c.logProject(ctx, session, r, audit.ActionCreated, p, nil, p)

	return c.withDSN(ctx, p), nil
}

// ListProjects lists the Insights projects of the repository.
func (c *Controller) ListProjects(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) ([]*types.InsightsProjectWithDSN, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	list, err := c.service.ListProjects(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	out := make([]*types.InsightsProjectWithDSN, len(list))
	for i, p := range list {
		out[i] = c.withDSN(ctx, p)
	}

	return out, nil
}
