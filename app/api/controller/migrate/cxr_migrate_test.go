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

package migrate

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const cxrRepoRef = "root/repo"

var errCxrBoom = errors.New("boom")

type cxrGitFake struct {
	git.Interface
	err error
	uid string
}

func (g cxrGitFake) CreateRepository(
	context.Context, *git.CreateRepositoryParams,
) (*git.CreateRepositoryOutput, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &git.CreateRepositoryOutput{UID: g.uid}, nil
}

type cxrURLs struct {
	url.Provider
}

func (cxrURLs) GetInternalAPIURL(context.Context) string { return "http://localhost:3000" }

func (cxrURLs) GenerateGITCloneURL(_ context.Context, path string) string {
	return "https://git.example.com/" + path + ".git"
}

func (cxrURLs) GenerateGITCloneSSHURL(_ context.Context, path string) string {
	return "ssh://git.example.com/" + path + ".git"
}

type cxrPublicAccess struct {
	publicaccess.Service
	supported    bool
	supportedErr error
	setErr       error
	setValue     *bool
}

func (p *cxrPublicAccess) IsPublicAccessSupported(
	context.Context, enum.PublicResourceType, string,
) (bool, error) {
	return p.supported, p.supportedErr
}

func (p *cxrPublicAccess) Set(_ context.Context, _ enum.PublicResourceType, _ string, enable bool) error {
	p.setValue = &enable
	return p.setErr
}

type cxrLimiter struct {
	countErr   error
	storageErr error
}

func (l cxrLimiter) RepoCount(context.Context, int64, int) error   { return l.countErr }
func (cxrLimiter) RepoSize(context.Context, int64) error           { return nil }
func (l cxrLimiter) RootSpaceStorage(context.Context, int64) error { return l.storageErr }

type cxrEnv struct {
	*cxrBase
	repo   *types.Repository
	git    cxrGitFake
	public *cxrPublicAccess
	lim    cxrLimiter
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	repo := b.createRepo(ctx, t, b.createSpace(ctx, t, "root"), "repo")
	return &cxrEnv{
		cxrBase: b,
		repo:    repo,
		git:     cxrGitFake{uid: "git-new"},
		public:  &cxrPublicAccess{supported: true},
	}
}

func (e *cxrEnv) controller(t *testing.T, mode cxrAuthMode) *Controller {
	t.Helper()
	sys, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "cxr", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	reporter, err := repoevents.NewReporter(sys)
	require.NoError(t, err)

	tx := dbtx.New(e.db)
	pullreqImporter := migrate.NewPullReq(
		nil, nil, e.principals, e.spaces, e.repos, nil, nil, nil, nil, nil, nil, nil, e.repoFinder, tx, nil)
	ruleStore := database.NewRuleStore(e.db, cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(e.db)))
	ruleImporter := migrate.NewRule(ruleStore, tx, e.principals)
	webhookImporter := migrate.NewWebhook(webhook.Config{}, tx, database.NewWebhookStore(e.db))
	labelImporter := migrate.NewLabel(
		database.NewLabelStore(e.db), database.NewLabelValueStore(e.db), e.spaces, tx)

	return NewController(
		cxrAuthorizer{mode: mode}, e.public, e.git, cxrURLs{}, pullreqImporter, ruleImporter, webhookImporter,
		labelImporter, e.lim, audit.New(), check.RepoIdentifierDefault, tx, e.spaces, e.repos, e.spaceFinder,
		e.repoFinder, reporter)
}

func (e *cxrEnv) setState(ctx context.Context, t *testing.T, state enum.RepoState) {
	t.Helper()
	_, err := e.db.ExecContext(ctx, "UPDATE repositories SET repo_state = $1 WHERE repo_id = $2",
		int(state), e.repo.ID)
	require.NoError(t, err)
}

func TestCxrMigrateCreateRepo(t *testing.T) {
	ctx := t.Context()
	t.Run("creates a migrating private repository", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		cxrBootstrap(ctx, t, e.cxrBase)
		out, err := e.controller(t, cxrAllow).CreateRepo(ctx, e.session(), &CreateRepoInput{
			ParentRef: "root", Identifier: "imported", DefaultBranch: "main", IsPublic: true,
		})
		require.NoError(t, err)
		require.Equal(t, "imported", out.Identifier)
		require.Equal(t, enum.RepoStateMigrateGitPush, out.State)
		require.True(t, out.IsEmpty)
		require.True(t, out.IsPublic)
		require.Equal(t, "https://git.example.com/root/imported.git", out.GitURL)
		require.Equal(t, "ssh://git.example.com/root/imported.git", out.GitSSHURL)
		require.NotNil(t, e.public.setValue)
		require.True(t, *e.public.setValue)
	})

	t.Run("public access not supported makes the repository private", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		cxrBootstrap(ctx, t, e.cxrBase)
		e.public.supported = false
		out, err := e.controller(t, cxrAllow).CreateRepo(ctx, e.session(), &CreateRepoInput{
			ParentRef: "root", Identifier: "imported", DefaultBranch: "main", IsPublic: true,
		})
		require.NoError(t, err)
		require.False(t, out.IsPublic)
		require.False(t, *e.public.setValue)
	})

	t.Run("failures", func(t *testing.T) {
		in := func() *CreateRepoInput {
			return &CreateRepoInput{ParentRef: "root", Identifier: "imported", DefaultBranch: "main"}
		}
		tests := []struct {
			name    string
			prepare func(e *cxrEnv)
			mode    cxrAuthMode
			in      *CreateRepoInput
			want    string
		}{
			{"unknown parent", func(*cxrEnv) {}, cxrAllow,
				&CreateRepoInput{ParentRef: "nope", Identifier: "x"}, "failed to check auth in parent"},
			{"forbidden", func(*cxrEnv) {}, cxrDeny, in(), "failed to check auth in parent"},
			{"git failure", func(e *cxrEnv) { e.git.err = errCxrBoom }, cxrAllow, in(), "failed to create git repository"},
			{"repository count limit", func(e *cxrEnv) { e.lim.countErr = errCxrBoom }, cxrAllow, in(),
				"maximum number of repositories reached"},
			{"storage over limit", func(e *cxrEnv) {
				e.lim.storageErr = &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit}
			}, cxrAllow, in(), "failed to create a repo on db"},
			{"duplicate identifier", func(*cxrEnv) {}, cxrAllow,
				&CreateRepoInput{ParentRef: "root", Identifier: "repo", DefaultBranch: "main"},
				"failed to create a repo on db"},
			{"public access check failure", func(e *cxrEnv) { e.public.supportedErr = errCxrBoom }, cxrAllow, in(),
				"failed to check if public access is supported"},
			{"public access update failure", func(e *cxrEnv) { e.public.setErr = errCxrBoom }, cxrAllow, in(),
				"failed to set repo access mode"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := cxrNewEnv(ctx, t)
				cxrBootstrap(ctx, t, e.cxrBase)
				tt.prepare(e)
				_, err := e.controller(t, tt.mode).CreateRepo(ctx, e.session(), tt.in)
				require.ErrorContains(t, err, tt.want)
			})
		}
	})
}

func TestCxrMigrateImporters(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	e.setState(ctx, t, enum.RepoStateMigrateDataImport)
	c := e.controller(t, cxrAllow)

	labels, err := c.Labels(ctx, e.session(), "root", &LabelsInput{Labels: []*migrate.ExternalLabel{
		{Name: "bug", Color: "red", Description: "a bug"},
		{Name: "prio", Value: "high", Color: "blue"},
	}})
	require.NoError(t, err)
	require.NotEmpty(t, labels)

	hooks, err := c.Webhooks(ctx, e.session(), cxrRepoRef, &WebhooksInput{Webhooks: []*migrate.ExternalWebhook{{
		Identifier: "hook", Target: "https://hooks.example.com/x", Active: true,
		Events: []string{string(enum.WebhookTriggerBranchCreated)},
	}}})
	require.NoError(t, err)
	require.Len(t, hooks, 1)
	require.Equal(t, "hook", hooks[0].Identifier)

	_, err = c.Webhooks(ctx, e.session(), cxrRepoRef, &WebhooksInput{Webhooks: []*migrate.ExternalWebhook{{
		Identifier: "bad hook!", Target: "https://hooks.example.com/x",
	}}})
	require.ErrorContains(t, err, "failed to import webhooks")

	rules, err := c.Rules(ctx, e.session(), cxrRepoRef, &RulesInput{Type: migrate.ExternalRuleTypeBranch})
	require.NoError(t, err)
	require.Empty(t, rules)

	_, err = c.Rules(ctx, e.session(), cxrRepoRef, &RulesInput{
		Type: migrate.ExternalRuleTypeBranch, Rules: []*migrate.ExternalRule{{Identifier: "bad rule!"}},
	})
	require.ErrorContains(t, err, "failed to import rules")

	rules, err = c.Rules(ctx, e.session(), cxrRepoRef, &RulesInput{
		Type: migrate.ExternalRuleTypeBranch, Rules: []*migrate.ExternalRule{{
			Identifier: "r1", State: "active", Definition: []byte(`{}`), Pattern: []byte(`{"default":true}`),
		}},
	})
	require.NoError(t, err)
	require.Len(t, rules, 1)

	prs, err := c.PullRequests(ctx, e.session(), cxrRepoRef, &PullreqsInput{})
	require.NoError(t, err)
	require.Empty(t, prs)
}

func TestCxrMigrateImportersGuards(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(t, cxrAllow)

	// The repository is not in the data import state.
	_, err := c.Webhooks(ctx, e.session(), cxrRepoRef, &WebhooksInput{})
	require.ErrorIs(t, err, errInvalidRepoState)
	_, err = c.Rules(ctx, e.session(), cxrRepoRef, &RulesInput{})
	require.ErrorIs(t, err, errInvalidRepoState)
	_, err = c.PullRequests(ctx, e.session(), cxrRepoRef, &PullreqsInput{})
	require.ErrorIs(t, err, errInvalidRepoState)

	deny := e.controller(t, cxrDeny)
	_, err = deny.Webhooks(ctx, e.session(), cxrRepoRef, &WebhooksInput{})
	require.ErrorContains(t, err, "failed to acquire access to repo")
	_, err = deny.Rules(ctx, e.session(), cxrRepoRef, &RulesInput{})
	require.ErrorContains(t, err, "failed to acquire access to repo")
	_, err = deny.PullRequests(ctx, e.session(), cxrRepoRef, &PullreqsInput{})
	require.ErrorContains(t, err, "failed to acquire access to repo")
	_, err = deny.Labels(ctx, e.session(), "root", &LabelsInput{})
	require.ErrorContains(t, err, "failed to acquire access to space")
	_, err = c.Webhooks(ctx, e.session(), "", &WebhooksInput{})
	require.ErrorContains(t, err, "failed to acquire access to repo")
}

// cxrBootstrap initializes the system service principal used when creating migrated repositories.
func cxrBootstrap(ctx context.Context, t *testing.T, b *cxrBase) {
	t.Helper()
	svcCtrl := service.NewController(check.PrincipalUIDDefault, cxrAuthorizer{mode: cxrAllow}, b.principals)
	cfg := &types.Config{}
	cfg.Principal.System.UID = "harness"
	cfg.Principal.System.Email = "system@example.com"
	cfg.Principal.System.DisplayName = "Harness"
	require.NoError(t, bootstrap.SystemService(ctx, cfg, svcCtrl))
}
