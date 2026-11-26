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

package space

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	ruleevents "github.com/harness/gitness/app/events/rule"
	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/rules"
	spacesvc "github.com/harness/gitness/app/services/space"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errCtlPublicAccess = errors.New("public access failure")

// ctlPublicAccess is an in-memory publicaccess.Service.
type ctlPublicAccess struct {
	mu          sync.Mutex
	public      map[string]bool
	unsupported bool
	getErr      error
	setErr      error
	supportErr  error
}

var _ publicaccess.Service = (*ctlPublicAccess)(nil)

func (p *ctlPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, path string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.public[path], p.getErr
}

func (p *ctlPublicAccess) Set(_ context.Context, _ enum.PublicResourceType, path string, enable bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.setErr != nil {
		return p.setErr
	}
	p.public[path] = enable
	return nil
}

func (p *ctlPublicAccess) Delete(_ context.Context, _ enum.PublicResourceType, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.public, path)
	return nil
}

func (p *ctlPublicAccess) IsPublicAccessSupported(
	context.Context, enum.PublicResourceType, string,
) (bool, error) {
	return !p.unsupported, p.supportErr
}

type ctlSSE struct{}

func (ctlSSE) Publish(context.Context, int64, enum.SSEType, any) {}

func (ctlSSE) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

type ctlURLProvider struct {
	url.Provider
}

func (ctlURLProvider) GenerateGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.com/" + repoPath + ".git"
}

func (ctlURLProvider) GenerateGITCloneSSHURL(_ context.Context, repoPath string) string {
	return "ssh://git@git.example.com/" + repoPath + ".git"
}

type ctlEnv struct {
	*ctlBase
	ctrl          *Controller
	publicAccess  *ctlPublicAccess
	pipelines     store.PipelineStore
	executions    store.ExecutionStore
	secrets       store.SecretStore
	templates     store.TemplateStore
	connectors    store.ConnectorStore
	autolinkStore store.AutoLinkStore
	root          *types.Space
}

func ctlNewEnv(t *testing.T, nested bool) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)
	b.bootstrapSystemService(t)

	secretStore := database.NewSecretStore(b.db)
	pipelineStore := database.NewPipelineStore(b.db)
	executionStore := database.NewExecutionStore(b.db)
	templateStore := database.NewTemplateStore(b.db)
	connectorStore := database.NewConnectorStore(b.db, secretStore)
	autolinkStore := database.NewAutoLinkStore(b.db)
	pa := &ctlPublicAccess{public: map[string]bool{}}

	autolinkSvc := autolink.NewService(b.tx, b.spaces, b.repos, autolinkStore)
	spaceSvc := spacesvc.NewService(
		b.tx, nil, nil, b.repos, b.spaces, b.spacePaths, nil, nil, nil, b.spaceFinder, nil, nil, nil)

	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(b.db))
	spaceEvictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", nil)
	repoEvictor := cache.NewEvictor[*types.RepositoryCore]("ns", "repo", nil)
	repoIDCache := cache.NewRepoIDCache(context.Background(), b.repos, spaceEvictor, repoEvictor, 0)

	eventsSystem, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	ruleReporter, err := ruleevents.NewReporter(eventsSystem)
	require.NoError(t, err)
	protectionManager, err := protection.ProvideManager(database.NewRuleStore(b.db, pCache))
	require.NoError(t, err)
	rulesSvc := rules.NewService(
		b.tx, database.NewRuleStore(b.db, pCache), b.repos, b.spaces, protectionManager, audit.New(),
		instrument.Noop{}, pCache, nil, nil, ruleReporter, ctlSSE{}, rules.ProvideValidator(), repoIDCache,
	)
	labelSvc := label.New(
		b.tx, b.spaces, database.NewLabelStore(b.db), database.NewLabelValueStore(b.db),
		database.NewPullReqLabelStore(b.db), database.NewPullReqLabelSuggestionStore(b.db), pCache, b.spaceFinder,
	)

	cfg := &types.Config{NestedSpacesEnabled: nested}
	ctrl := NewController(
		cfg, b.tx, ctlURLProvider{}, ctlSSE{}, check.SpaceIdentifierDefault, ctlAuthorizer{mode: ctlAllow},
		b.spacePaths, pipelineStore, secretStore, connectorStore, templateStore, b.spaces, b.repos, b.principals,
		nil, b.memberships, nil, b.spaceFinder, b.repoFinder, nil, nil, limiter.NewResourceLimiter(), pa,
		audit.New(), nil, labelSvc, instrument.Noop{}, executionStore, rulesSvc,
		database.NewUsageMetricsStore(b.db), check.RepoIdentifierDefault, nil,
		database.NewFavoriteStore(b.db), autolinkSvc, spaceSvc,
	)

	e := &ctlEnv{
		ctlBase:       b,
		ctrl:          ctrl,
		publicAccess:  pa,
		pipelines:     pipelineStore,
		executions:    executionStore,
		secrets:       secretStore,
		templates:     templateStore,
		connectors:    connectorStore,
		autolinkStore: autolinkStore,
	}
	e.root = b.createSpace(t, "root", 0)
	return e
}
