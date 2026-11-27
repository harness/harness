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

package importer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

const sv5SystemPrincipalID = 77

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: sv5SystemPrincipalID, UID: "sys", Admin: true}, nil
}

// sv5Booted is set once at package initialization: the system service principal is package-level state.
var sv5Booted = sv5InitBootstrap()

func sv5InitBootstrap() bool {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys"
	ctrl := service.NewController(check.PrincipalUIDDefault, nil, sv5PrincipalStore{})
	if err := bootstrap.SystemService(context.Background(), cfg, ctrl); err != nil {
		panic(err)
	}
	return true
}

func sv5Bootstrap(t *testing.T) {
	t.Helper()
	require.True(t, sv5Booted)
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

type sv5Git struct {
	git.Interface

	mu          sync.Mutex
	createErr   error
	syncErr     error
	deleteErr   error
	syncs       []*git.SyncRepositoryParams
	deletes     int
	defaultName string
}

func (g *sv5Git) CreateRepository(context.Context, *git.CreateRepositoryParams) (*git.CreateRepositoryOutput, error) {
	if g.createErr != nil {
		return nil, g.createErr
	}
	return &git.CreateRepositoryOutput{UID: "git-uid-1"}, nil
}

func (g *sv5Git) SyncRepository(_ context.Context, p *git.SyncRepositoryParams) (*git.SyncRepositoryOutput, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.syncs = append(g.syncs, p)
	if g.syncErr != nil {
		return nil, g.syncErr
	}
	return &git.SyncRepositoryOutput{DefaultBranch: g.defaultName}, nil
}

func (g *sv5Git) DeleteRepository(context.Context, *git.DeleteRepositoryParams) error {
	g.deletes++
	return g.deleteErr
}

type sv5RepoStore struct {
	store.RepoStore

	repo    *types.Repository
	findErr error
	updates int
}

func (s *sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	cp := *s.repo
	return &cp, nil
}

func (s *sv5RepoStore) UpdateOptLock(
	_ context.Context, repo *types.Repository, fn func(*types.Repository) error,
) (*types.Repository, error) {
	s.updates++
	cp := *repo
	if err := fn(&cp); err != nil {
		return nil, err
	}
	*s.repo = cp
	return &cp, nil
}

type sv5SSE struct {
	sse.Streamer

	published []enum.SSEType
}

func (s *sv5SSE) Publish(_ context.Context, _ int64, t enum.SSEType, _ any) {
	s.published = append(s.published, t)
}

type sv5Indexer struct {
	err   error
	calls int
}

func (i *sv5Indexer) Index(context.Context, *types.Repository) error {
	i.calls++
	return i.err
}

type sv5PublicAccess struct {
	publicaccess.Service

	supported    bool
	supportedErr error
	setErr       error
	setTo        *bool
}

func (p *sv5PublicAccess) IsPublicAccessSupported(context.Context, enum.PublicResourceType, string) (bool, error) {
	return p.supported, p.supportedErr
}

func (p *sv5PublicAccess) Set(_ context.Context, _ enum.PublicResourceType, _ string, enable bool) error {
	p.setTo = &enable
	return p.setErr
}

type sv5Audit struct {
	err   error
	calls int
}

func (a *sv5Audit) Log(
	context.Context, types.Principal, audit.Resource, audit.Action, string, ...audit.Option,
) error {
	a.calls++
	return a.err
}

type sv5SettingsStore struct {
	store.SettingsStore

	err error
}

func (s sv5SettingsStore) Upsert(context.Context, enum.SettingsScope, int64, string, json.RawMessage) error {
	return s.err
}

type sv5RepoIDCache struct {
	store.RepoIDCache

	repos map[int64]*types.RepositoryCore
}

func (c sv5RepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	r, ok := c.repos[id]
	if !ok {
		return nil, errSv5Boom
	}
	return r, nil
}

func sv5RepoFinder(repos ...*types.RepositoryCore) refcache.RepoFinder {
	m := map[int64]*types.RepositoryCore{}
	for _, r := range repos {
		m[r.ID] = r
	}
	return refcache.NewRepoFinder(nil, nil, sv5RepoIDCache{repos: m}, nil, cache.Evictor[*types.RepositoryCore]{})
}

type sv5Producer struct{}

func (sv5Producer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func sv5Reporter(t *testing.T) *repoevents.Reporter {
	t.Helper()
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSv5Boom },
		sv5Producer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	reporter, err := repoevents.NewReporter(system)
	require.NoError(t, err)
	return reporter
}

func sv5ImportingRepo() *types.Repository {
	return &types.Repository{
		ID: 5, ParentID: 9, Identifier: "repo", Path: "space/repo",
		State: enum.RepoStateGitImport, DefaultBranch: "main",
	}
}

type sv5ImportEnv struct {
	importer *Importer
	git      *sv5Git
	repos    *sv5RepoStore
	sse      *sv5SSE
	indexer  *sv5Indexer
	access   *sv5PublicAccess
	audit    *sv5Audit
}

func sv5NewImportEnv(t *testing.T) *sv5ImportEnv {
	t.Helper()
	sv5Bootstrap(t)
	env := &sv5ImportEnv{
		git:     &sv5Git{defaultName: "trunk"},
		repos:   &sv5RepoStore{repo: sv5ImportingRepo()},
		sse:     &sv5SSE{},
		indexer: &sv5Indexer{},
		access:  &sv5PublicAccess{supported: true},
		audit:   &sv5Audit{},
	}
	env.importer = NewImporter(
		"main", sv5URLProvider{}, env.git, nil, env.repos, nil, nil, sv5RepoFinder(),
		env.sse, env.indexer, env.access, sv5Reporter(t), env.audit,
		settings.NewService(sv5SettingsStore{}, refcache.SpaceFinder{}),
		netpolicy.Policy{}, false,
	)
	return env
}

func sv5Input() Input {
	return Input{
		RepoID: 5, Public: true, GitUser: "user", GitPass: "pass",
		CloneURL: "https://1.1.1.1/org/repo.git", Pipelines: PipelineOptionIgnore,
	}
}

func TestSv5ImporterImportSuccess(t *testing.T) {
	env := sv5NewImportEnv(t)

	require.NoError(t, env.importer.Import(context.Background(), sv5Input()))

	require.Equal(t, enum.RepoStateActive, env.repos.repo.State)
	require.Equal(t, "git-uid-1", env.repos.repo.GitUID)
	require.Equal(t, "trunk", env.repos.repo.DefaultBranch)
	require.Len(t, env.git.syncs, 1)
	require.Equal(t, "https://user:pass@1.1.1.1/org/repo.git", env.git.syncs[0].Source)
	require.NotNil(t, env.access.setTo)
	require.True(t, *env.access.setTo)
	require.Equal(t, 1, env.audit.calls)
	require.Equal(t, 1, env.indexer.calls)
	require.Equal(t, []enum.SSEType{enum.SSETypeRepositoryImportCompleted}, env.sse.published)
}

func TestSv5ImporterImportPrivateWhenUnsupported(t *testing.T) {
	env := sv5NewImportEnv(t)
	env.access.supported = false
	env.audit.err = errSv5Boom
	env.indexer.err = errSv5Boom

	require.NoError(t, env.importer.Import(context.Background(), sv5Input()))
	require.False(t, *env.access.setTo)
	require.Zero(t, env.audit.calls)
}

func TestSv5ImporterImportAuditFailureIsTolerated(t *testing.T) {
	env := sv5NewImportEnv(t)
	env.audit.err = errSv5Boom

	require.NoError(t, env.importer.Import(context.Background(), sv5Input()))
	require.Equal(t, 1, env.audit.calls)
}

func TestSv5ImporterImportPreconditionErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		mutate func(env *sv5ImportEnv, in *Input)
		msg    string
	}{
		{"missing clone url", func(_ *sv5ImportEnv, in *Input) { in.CloneURL = "" }, "missing git repository clone URL"},
		{"unparsable clone url", func(_ *sv5ImportEnv, in *Input) { in.CloneURL = "://x" }, "parse git clone URL"},
		{"disallowed clone url", func(_ *sv5ImportEnv, in *Input) { in.CloneURL = "http://127.0.0.1/r.git" },
			"clone URL of the provider is not allowed"},
		{"ssh clone url", func(_ *sv5ImportEnv, in *Input) { in.CloneURL = "ssh://1.1.1.1/r.git" },
			"clone URL of the provider is not allowed"},
		{"find repo", func(env *sv5ImportEnv, _ *Input) { env.repos.findErr = errSv5Boom }, "find repo by id"},
		{"repo not importing", func(env *sv5ImportEnv, _ *Input) { env.repos.repo.State = enum.RepoStateActive },
			"is not being imported"},
		{"public access support", func(env *sv5ImportEnv, _ *Input) { env.access.supportedErr = errSv5Boom },
			"public access is supported"},
		{"public access set", func(env *sv5ImportEnv, _ *Input) { env.access.setErr = errSv5Boom },
			"set repo access mode"},
		{"create git repo", func(env *sv5ImportEnv, _ *Input) { env.git.createErr = errSv5Boom },
			"create empty git repository"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := sv5NewImportEnv(t)
			in := sv5Input()
			tc.mutate(env, &in)
			require.ErrorContains(t, env.importer.Import(ctx, in), tc.msg)
			require.Empty(t, env.sse.published)
		})
	}
}

func TestSv5ImporterImportSyncFailureCleansUp(t *testing.T) {
	env := sv5NewImportEnv(t)
	env.git.syncErr = errSv5Boom

	err := env.importer.Import(context.Background(), sv5Input())
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "failed to import repository")

	require.Equal(t, 1, env.git.deletes)
	require.Equal(t, enum.RepoStateImportFailed, env.repos.repo.State)
	require.Equal(t, []enum.SSEType{enum.SSETypeRepositoryImportCompleted}, env.sse.published)
	require.Zero(t, env.indexer.calls)
}

func TestSv5JobRepositoryHandle(t *testing.T) {
	env := sv5NewImportEnv(t)
	jr := &JobRepository{encrypter: sv5Crypt{}, importer: env.importer}

	in := RepositoryInput{Input: sv5Input()}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	data := base64.StdEncoding.EncodeToString(raw)

	res, err := jr.Handle(context.Background(), data, nil)
	require.NoError(t, err)
	require.Empty(t, res)
	require.Equal(t, enum.RepoStateActive, env.repos.repo.State)

	// A second run finds the repository already imported.
	_, err = jr.Handle(context.Background(), data, nil)
	require.ErrorContains(t, err, "failed to import repository")

	_, err = jr.Handle(context.Background(), "!!!", nil)
	require.Error(t, err)
}

type sv5Crypt struct{}

func (sv5Crypt) Encrypt(p string) ([]byte, error) { return []byte(p), nil }
func (sv5Crypt) Decrypt(c []byte) (string, error) { return string(c), nil }
