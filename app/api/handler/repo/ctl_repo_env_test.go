// Copyright 2026 Harness, Inc.
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
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	repoctrl "github.com/harness/gitness/app/api/controller/repo"
	repoevents "github.com/harness/gitness/app/events/repo"
	ruleevents "github.com/harness/gitness/app/events/rule"
	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	gitmock "github.com/harness/gitness/mocks/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// ctlPublicAccess is an in-memory publicaccess.Service.
type ctlPublicAccess struct {
	mu     sync.Mutex
	public map[string]bool
}

var _ publicaccess.Service = (*ctlPublicAccess)(nil)

func (p *ctlPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, path string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.public[path], nil
}

func (p *ctlPublicAccess) Set(_ context.Context, _ enum.PublicResourceType, path string, enable bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.public[path] = enable
	return nil
}

func (p *ctlPublicAccess) Delete(_ context.Context, _ enum.PublicResourceType, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.public, path)
	return nil
}

func (p *ctlPublicAccess) IsPublicAccessSupported(context.Context, enum.PublicResourceType, string) (bool, error) {
	return true, nil
}

type ctlSSE struct{}

func (ctlSSE) Publish(context.Context, int64, enum.SSEType, any) {}

func (ctlSSE) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

type ctlURLProvider struct {
	url.Provider
}

func (ctlURLProvider) GetInternalAPIURL(context.Context) string { return "http://localhost:3000" }

func (ctlURLProvider) GenerateGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.com/" + repoPath + ".git"
}

func (ctlURLProvider) GenerateGITCloneSSHURL(_ context.Context, repoPath string) string {
	return "ssh://git@git.example.com/" + repoPath + ".git"
}

// ctlUserGroupService is a usergroup.Service without any user groups.
type ctlUserGroupService struct{}

func (ctlUserGroupService) List(context.Context, *types.ListQueryFilter, *types.SpaceCore) (
	[]*types.UserGroupInfo, error,
) {
	return []*types.UserGroupInfo{}, nil
}

func (ctlUserGroupService) ListUserIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return []int64{}, nil
}

func (ctlUserGroupService) MapGroupIDsToPrincipals(
	context.Context, []int64,
) (map[int64][]*types.Principal, error) {
	return map[int64][]*types.Principal{}, nil
}

type ctlEnv struct {
	*ctlBase
	ctrl         *repoctrl.Controller
	git          *gitmock.Interface
	publicAccess *ctlPublicAccess
	settings     *settings.Service
	space        *types.Space
	repo         *types.Repository
}

func ctlNewEnv(t *testing.T, mode ctlAuthMode) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)
	b.bootstrapSystemService(t)

	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(b.db))
	spaceEvictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", nil)
	repoEvictor := cache.NewEvictor[*types.RepositoryCore]("ns", "repo", nil)
	repoIDCache := cache.NewRepoIDCache(t.Context(), b.repos, spaceEvictor, repoEvictor, 0)

	eventsSystem, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	ruleReporter, err := ruleevents.NewReporter(eventsSystem)
	require.NoError(t, err)
	repoReporter, err := repoevents.NewReporter(eventsSystem)
	require.NoError(t, err)

	ruleStore := database.NewRuleStore(b.db, pCache)
	protectionManager, err := protection.ProvideManager(ruleStore)
	require.NoError(t, err)
	rulesSvc := rules.NewService(
		b.tx, ruleStore, b.repos, b.spaces, protectionManager, audit.New(), instrument.Noop{}, pCache,
		nil, nil, ruleReporter, ctlSSE{}, rules.ProvideValidator(), repoIDCache,
	)
	labelSvc := label.New(
		b.tx, b.spaces, database.NewLabelStore(b.db), database.NewLabelValueStore(b.db),
		database.NewPullReqLabelStore(b.db), database.NewPullReqLabelSuggestionStore(b.db), pCache, b.spaceFinder,
	)
	autolinkSvc := autolink.NewService(b.tx, b.spaces, b.repos, database.NewAutoLinkStore(b.db))
	settingsSvc := settings.NewService(database.NewSettingsStore(b.db), b.spaceFinder)
	pa := &ctlPublicAccess{public: map[string]bool{}}
	gitMock := &gitmock.Interface{}

	cfg := &types.Config{}
	cfg.Git.DefaultBranch = "main"

	ctrl := repoctrl.NewController(
		cfg, b.tx, ctlURLProvider{}, ctlAuthorizer{mode: mode}, b.repos,
		database.NewRepoActivityStore(b.db), database.NewLinkedRepoStore(b.db), b.spaces,
		database.NewPipelineStore(b.db), database.NewExecutionStore(b.db), b.principals, ruleStore,
		database.NewCheckStore(b.db, pCache), database.NewPullReqStore(b.db, pCache), settingsSvc, pCache,
		protectionManager, gitMock, b.spaceFinder, b.repoFinder, nil, nil, nil, nil, nil, repoReporter, nil,
		limiter.NewResourceLimiter(), nil, audit.New(), nil, check.RepoIdentifierDefault,
		repoctrl.NewNoOpRepoChecks(), pa, labelSvc, instrument.Noop{}, database.NewUserGroupStore(b.db),
		ctlUserGroupService{}, rulesSvc, ctlSSE{}, nil, database.NewFavoriteStore(b.db),
		publickey.SignatureVerifyService{}, autolinkSvc, nil, nil, nil, database.NewRepoLangStore(b.db), nil,
	)

	e := &ctlEnv{ctlBase: b, ctrl: ctrl, git: gitMock, publicAccess: pa, settings: settingsSvc}
	e.space = b.createSpace(t, "root", 0)
	e.repo = b.createRepo(t, e.space, "repo1")
	return e
}
