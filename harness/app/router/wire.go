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

package router

import (
	"context"
	"strings"

	"github.com/harness/gitness/app/api/controller/check"
	"github.com/harness/gitness/app/api/controller/connector"
	"github.com/harness/gitness/app/api/controller/execution"
	"github.com/harness/gitness/app/api/controller/githook"
	"github.com/harness/gitness/app/api/controller/gitspace"
	"github.com/harness/gitness/app/api/controller/infraprovider"
	"github.com/harness/gitness/app/api/controller/keywordsearch"
	"github.com/harness/gitness/app/api/controller/lfs"
	"github.com/harness/gitness/app/api/controller/logs"
	"github.com/harness/gitness/app/api/controller/migrate"
	"github.com/harness/gitness/app/api/controller/pipeline"
	"github.com/harness/gitness/app/api/controller/plugin"
	"github.com/harness/gitness/app/api/controller/principal"
	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/controller/reposettings"
	"github.com/harness/gitness/app/api/controller/secret"
	"github.com/harness/gitness/app/api/controller/serviceaccount"
	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/controller/template"
	"github.com/harness/gitness/app/api/controller/trigger"
	"github.com/harness/gitness/app/api/controller/upload"
	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/controller/usergroup"
	"github.com/harness/gitness/app/api/controller/webhook"
	"github.com/harness/gitness/app/api/openapi"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/services/usage"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/registry/app/api"
	"github.com/harness/gitness/registry/app/api/router"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideRouter,
	api.WireSet,
)

func GetGitRoutingHost(ctx context.Context, urlProvider url.Provider) string {
	// use url provider as it has the latest data.
	gitHostname := urlProvider.GetGITHostname(ctx)
	apiHostname := urlProvider.GetAPIHostname(ctx)

	// only use host name to identify git traffic if it differs from api hostname.
	// TODO: Can we make this even more flexible - aka use the full base urls to route traffic?
	gitRoutingHost := ""
	if !strings.EqualFold(gitHostname, apiHostname) {
		gitRoutingHost = gitHostname
	}
	return gitRoutingHost
}

// ProvideRouter provides ordered list of routers.
func ProvideRouter(
	appCtx context.Context,
	config *types.Config,
	authenticator authn.Authenticator,
	repoCtrl *repo.Controller,
	repoSettingsCtrl *reposettings.Controller,
	executionCtrl *execution.Controller,
	logCtrl *logs.Controller,
	spaceCtrl *space.Controller,
	pipelineCtrl *pipeline.Controller,
	secretCtrl *secret.Controller,
	triggerCtrl *trigger.Controller,
	connectorCtrl *connector.Controller,
	templateCtrl *template.Controller,
	pluginCtrl *plugin.Controller,
	pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller,
	githookCtrl *githook.Controller,
	git git.Interface,
	saCtrl *serviceaccount.Controller,
	userCtrl *user.Controller,
	principalCtrl principal.Controller,
	userGroupCtrl *usergroup.Controller,
	checkCtrl *check.Controller,
	sysCtrl *system.Controller,
	blobCtrl *upload.Controller,
	searchCtrl *keywordsearch.Controller,
	infraProviderCtrl *infraprovider.Controller,
	gitspaceCtrl *gitspace.Controller,
	migrateCtrl *migrate.Controller,
	urlProvider url.Provider,
	openapi openapi.Service,
	registryRouter router.AppRouter,
	usageSender usage.Sender,
	lfsCtrl *lfs.Controller,
) *Router {
	routers := make([]Interface, 4)

	gitRoutingHost := GetGitRoutingHost(appCtx, urlProvider)
	gitHandler := NewGitHandler(
		config,
		urlProvider,
		authenticator,
		repoCtrl,
		usageSender,
		lfsCtrl,
	)
	routers[0] = NewGitRouter(gitHandler, gitRoutingHost)
	routers[1] = router.NewRegistryRouter(registryRouter)

	apiHandler := NewAPIHandler(
		appCtx, config,
		authenticator, repoCtrl, repoSettingsCtrl, executionCtrl, logCtrl, spaceCtrl, pipelineCtrl,
		secretCtrl, triggerCtrl, connectorCtrl, templateCtrl, pluginCtrl, pullreqCtrl, webhookCtrl,
		githookCtrl, git, saCtrl, userCtrl, principalCtrl, userGroupCtrl, checkCtrl, sysCtrl, blobCtrl, searchCtrl,
		infraProviderCtrl, migrateCtrl, gitspaceCtrl, usageSender)
	routers[2] = NewAPIRouter(apiHandler)

	sec := NewSecure(config)
	webHandler := NewWebHandler(
		authenticator, openapi, sec,
		config.PublicResourceCreationEnabled,
		config.Development.UISourceOverride,
	)
	routers[3] = NewWebRouter(webHandler)

	return NewRouter(routers)
}
