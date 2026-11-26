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

package languageanalyzer

import (
	"context"
	"errors"
	"sort"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/langstats"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbTx struct{}

func (svbTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

var _ dbtx.Transactor = svbTx{}

type svbRepoStore struct {
	store.RepoStore

	repo      *types.Repository
	findErr   error
	updateErr error
	updated   *types.Repository
	findCalls int
}

func (f *svbRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	f.findCalls++
	return f.repo, f.findErr
}

func (f *svbRepoStore) UpdateOptLock(
	_ context.Context, repo *types.Repository, mutate func(*types.Repository) error,
) (*types.Repository, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if err := mutate(repo); err != nil {
		return nil, err
	}
	f.updated = repo
	return repo, nil
}

type svbLangStore struct {
	store.RepoLangStore

	deleteErr error
	insertErr error
	deleted   bool
	inserted  []*types.RepoLangStat
}

func (f *svbLangStore) DeleteByRepoID(context.Context, int64) error {
	f.deleted = true
	return f.deleteErr
}

func (f *svbLangStore) InsertByRepoID(_ context.Context, _ int64, rows []*types.RepoLangStat) error {
	f.inserted = rows
	return f.insertErr
}

type svbGit struct {
	git.Interface

	out    git.GetRepoLanguageStatsOutput
	err    error
	params *git.GetRepoLanguageStatsParams
}

func (g *svbGit) GetRepoLanguageStats(
	_ context.Context, p *git.GetRepoLanguageStatsParams,
) (git.GetRepoLanguageStatsOutput, error) {
	g.params = p
	return g.out, g.err
}

type svbRepoIDCache struct {
	store.RepoIDCache

	repo *types.RepositoryCore
	err  error
}

func (f *svbRepoIDCache) Get(context.Context, int64) (*types.RepositoryCore, error) {
	return f.repo, f.err
}

type svbEnv struct {
	analyzer Analyzer
	repos    *svbRepoStore
	langs    *svbLangStore
	git      *svbGit
	cache    *svbRepoIDCache
}

func svbNewEnv() *svbEnv {
	env := &svbEnv{
		repos: &svbRepoStore{repo: &types.Repository{ID: 1, GitUID: "uid", DefaultBranch: "main"}},
		langs: &svbLangStore{},
		git: &svbGit{out: git.GetRepoLanguageStatsOutput{
			LOC: 1234,
			Stats: map[string]*langstats.LangStat{
				"Go":                   {Bytes: 100, Files: 3},
				"Python":               {Bytes: 100, Files: 1},
				"Shell":                {Bytes: 10, Files: 2},
				"":                     {Bytes: 999, Files: 9},
				langstats.Unclassified: {Bytes: 500, Files: 5},
			},
		}},
		cache: &svbRepoIDCache{repo: &types.RepositoryCore{ID: 1, DefaultBranch: "main"}},
	}
	env.analyzer = NewAnalyzer(
		svbTx{}, env.repos,
		refcache.NewRepoFinder(nil, nil, env.cache, nil, cache.Evictor[*types.RepositoryCore]{}),
		env.langs, env.git,
	)
	return env
}

func TestSvbAnalyzeLanguages(t *testing.T) {
	ctx := context.Background()

	t.Run("stores stats and main language", func(t *testing.T) {
		env := svbNewEnv()
		require.NoError(t, env.analyzer.AnalyzeLanguages(ctx, 1))

		require.Equal(t, "uid", env.git.params.RepoUID)
		require.Equal(t, "main", env.git.params.Branch)
		require.True(t, env.langs.deleted)

		got := make([]string, 0, len(env.langs.inserted))
		for _, row := range env.langs.inserted {
			got = append(got, row.Language)
		}
		sort.Strings(got)
		require.Equal(t, []string{"Go", "Python", "Shell"}, got, "empty and unclassified languages are dropped")

		// Go and Python tie on bytes; the lexicographically smaller one wins.
		require.Equal(t, "Go", env.repos.updated.Language)
		require.Equal(t, int64(1234), env.repos.updated.LOC)
	})

	t.Run("failures", func(t *testing.T) {
		env := svbNewEnv()
		env.repos.findErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.AnalyzeLanguages(ctx, 1), errSvbBoom)

		env = svbNewEnv()
		env.git.err = errSvbBoom
		require.ErrorIs(t, env.analyzer.AnalyzeLanguages(ctx, 1), errSvbBoom)

		env = svbNewEnv()
		env.langs.deleteErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.AnalyzeLanguages(ctx, 1), errSvbBoom)

		env = svbNewEnv()
		env.langs.insertErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.AnalyzeLanguages(ctx, 1), errSvbBoom)

		env = svbNewEnv()
		env.repos.updateErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.AnalyzeLanguages(ctx, 1), errSvbBoom)
	})
}

func TestSvbEventHandlers(t *testing.T) {
	ctx := context.Background()

	t.Run("default branch updated", func(t *testing.T) {
		env := svbNewEnv()
		ev := &events.Event[*repoevents.DefaultBranchUpdatedPayload]{
			Payload: &repoevents.DefaultBranchUpdatedPayload{Base: repoevents.Base{RepoID: 1}},
		}
		require.NoError(t, env.analyzer.defaultBranchUpdatedHandler(ctx, ev))
		require.Equal(t, 1, env.repos.findCalls)

		env.repos.findErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.defaultBranchUpdatedHandler(ctx, ev), errSvbBoom)
	})

	t.Run("imported repo created", func(t *testing.T) {
		env := svbNewEnv()
		native := &events.Event[*repoevents.CreatedPayload]{
			Payload: &repoevents.CreatedPayload{Base: repoevents.Base{RepoID: 1}},
		}
		require.NoError(t, env.analyzer.importedRepoCreatedHandler(ctx, native))
		require.Zero(t, env.repos.findCalls, "native repos are not analyzed")

		imported := &events.Event[*repoevents.CreatedPayload]{
			Payload: &repoevents.CreatedPayload{Base: repoevents.Base{RepoID: 1}, ImportedFrom: "github"},
		}
		require.NoError(t, env.analyzer.importedRepoCreatedHandler(ctx, imported))
		require.Equal(t, 1, env.repos.findCalls)

		env.repos.findErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.importedRepoCreatedHandler(ctx, imported), errSvbBoom)
	})

	t.Run("branch created", func(t *testing.T) {
		env := svbNewEnv()
		mk := func(ref string) *events.Event[*gitevents.BranchCreatedPayload] {
			return &events.Event[*gitevents.BranchCreatedPayload]{
				Payload: &gitevents.BranchCreatedPayload{RepoID: 1, Ref: ref},
			}
		}
		require.NoError(t, env.analyzer.handleEventBranchCreated(ctx, mk("refs/heads/dev")))
		require.Zero(t, env.repos.findCalls, "non default branches are ignored")

		require.NoError(t, env.analyzer.handleEventBranchCreated(ctx, mk("refs/heads/main")))
		require.Equal(t, 1, env.repos.findCalls)

		env.repos.findErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.handleEventBranchCreated(ctx, mk("refs/heads/main")), errSvbBoom)

		env.cache.err = errSvbBoom
		require.ErrorIs(t, env.analyzer.handleEventBranchCreated(ctx, mk("refs/heads/main")), errSvbBoom)
	})

	t.Run("branch updated", func(t *testing.T) {
		env := svbNewEnv()
		mk := func(ref string) *events.Event[*gitevents.BranchUpdatedPayload] {
			return &events.Event[*gitevents.BranchUpdatedPayload]{
				Payload: &gitevents.BranchUpdatedPayload{RepoID: 1, Ref: ref},
			}
		}
		require.NoError(t, env.analyzer.handleEventBranchUpdated(ctx, mk("refs/heads/dev")))
		require.Zero(t, env.repos.findCalls)

		require.NoError(t, env.analyzer.handleEventBranchUpdated(ctx, mk("refs/heads/main")))
		require.Equal(t, 1, env.repos.findCalls)

		env.repos.findErr = errSvbBoom
		require.ErrorIs(t, env.analyzer.handleEventBranchUpdated(ctx, mk("refs/heads/main")), errSvbBoom)

		env.cache.err = errSvbBoom
		require.ErrorIs(t, env.analyzer.handleEventBranchUpdated(ctx, mk("refs/heads/main")), errSvbBoom)
	})
}

type svbConsumer struct{ registered int }

func (c *svbConsumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error {
	c.registered++
	return nil
}

func (c *svbConsumer) Configure(...stream.ConsumerOption) {}

func (c *svbConsumer) Start(context.Context) error { return nil }

func (c *svbConsumer) Errors() <-chan error { return make(chan error) }

func (c *svbConsumer) Infos() <-chan string { return make(chan string) }

type svbProducer struct{}

func (svbProducer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func svbSystem(t *testing.T, consumer *svbConsumer, failAfter int) *events.System {
	t.Helper()
	calls := 0
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) {
			calls++
			if calls > failAfter {
				return nil, errSvbBoom
			}
			return consumer, nil
		},
		svbProducer{},
		events.NewNoopCollector(),
	)
	require.NoError(t, err)
	return system
}

func TestSvbRegisterEventListeners(t *testing.T) {
	ctx := context.Background()
	cfg := &types.Config{}
	cfg.LanguageAnalyzer.RepoEventsConcurrency = 1
	cfg.LanguageAnalyzer.GitEventsConcurrency = 1

	run := func(t *testing.T, system *events.System) error {
		t.Helper()
		repoFactory, err := repoevents.NewReaderFactory(system)
		require.NoError(t, err)
		gitFactory, err := gitevents.NewReaderFactory(system)
		require.NoError(t, err)
		return RegisterEventListeners(ctx, "inst", repoFactory, gitFactory, svbNewEnv().analyzer, cfg)
	}

	consumer := &svbConsumer{}
	require.NoError(t, run(t, svbSystem(t, consumer, 2)))
	require.Equal(t, 4, consumer.registered)

	require.ErrorContains(t, run(t, svbSystem(t, &svbConsumer{}, 0)), "repo event reader")
	require.ErrorContains(t, run(t, svbSystem(t, &svbConsumer{}, 1)), "git event reader")
}
