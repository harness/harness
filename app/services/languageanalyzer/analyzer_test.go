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

package languageanalyzer

import (
	"context"
	"errors"
	"testing"

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/langstats"
	mockgit "github.com/harness/gitness/mocks/git"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeTx struct{ err error }

func (f fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

type fakeLangStore struct {
	store.RepoLangStore

	deleted     []int64
	inserted    [][]*types.RepoLangStat
	deleteErr   error
	insertErr   error
	callOrdered []string
}

func (f *fakeLangStore) DeleteByRepoID(_ context.Context, repoID int64) error {
	f.callOrdered = append(f.callOrdered, "delete")
	f.deleted = append(f.deleted, repoID)
	return f.deleteErr
}

func (f *fakeLangStore) InsertByRepoID(_ context.Context, _ int64, langs []*types.RepoLangStat) error {
	f.callOrdered = append(f.callOrdered, "insert")
	f.inserted = append(f.inserted, langs)
	return f.insertErr
}

type fixture struct {
	repo      *types.Repository
	repoStore *mockstore.RepoStore
	langStore *fakeLangStore
	git       *mockgit.Interface
	analyzer  Analyzer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		repo:      &types.Repository{ID: 3, GitUID: "git-uid", DefaultBranch: "main"},
		repoStore: &mockstore.RepoStore{},
		langStore: &fakeLangStore{},
		git:       &mockgit.Interface{},
	}
	f.analyzer = NewAnalyzer(fakeTx{}, f.repoStore, refcache.RepoFinder{}, f.langStore, f.git)
	t.Cleanup(func() {
		f.repoStore.AssertExpectations(t)
		f.git.AssertExpectations(t)
	})
	return f
}

func (f *fixture) expectStats(out git.GetRepoLanguageStatsOutput, err error) {
	f.repoStore.On("Find", f.repo.ID).Return(f.repo, nil)
	f.git.On("GetRepoLanguageStats", mock.Anything, &git.GetRepoLanguageStatsParams{
		ReadParams: git.ReadParams{RepoUID: "git-uid"},
		Branch:     "main",
	}).Return(out, err)
}

// expectUpdate expects the optimistic lock update and applies the mutation to a copy of the repo.
func (f *fixture) expectUpdate(updated *types.Repository) {
	f.repoStore.On("UpdateOptLock", f.repo, mock.Anything).
		Run(func(args mock.Arguments) {
			fn, ok := args.Get(1).(func(*types.Repository) error)
			if !ok {
				panic("unexpected mutate function type")
			}
			if err := fn(updated); err != nil {
				panic(err)
			}
		}).
		Return(updated, nil)
}

func TestAnalyzeLanguages(t *testing.T) {
	t.Run("picks the biggest language and stores all of them", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{
			LOC: 1234,
			Stats: map[string]*langstats.LangStat{
				"Go":                   {Bytes: 900, Files: 9},
				"TypeScript":           {Bytes: 400, Files: 4},
				"":                     {Bytes: 100000, Files: 1},
				langstats.Unclassified: {Bytes: 100000, Files: 1},
			},
		}, nil)
		updated := &types.Repository{}
		f.expectUpdate(updated)

		require.NoError(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID))

		require.Equal(t, []int64{3}, f.langStore.deleted)
		require.Equal(t, []string{"delete", "insert"}, f.langStore.callOrdered)
		require.Len(t, f.langStore.inserted, 1)
		require.ElementsMatch(t, []*types.RepoLangStat{
			{Language: "Go", Bytes: 900, Files: 9},
			{Language: "TypeScript", Bytes: 400, Files: 4},
		}, f.langStore.inserted[0])

		require.Equal(t, "Go", updated.Language)
		require.Equal(t, int64(1234), updated.LOC)
	})

	t.Run("ties are broken lexicographically", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{
			Stats: map[string]*langstats.LangStat{
				"Rust": {Bytes: 500, Files: 1},
				"C":    {Bytes: 500, Files: 1},
				"Zig":  {Bytes: 500, Files: 1},
			},
		}, nil)
		updated := &types.Repository{}
		f.expectUpdate(updated)

		require.NoError(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID))
		require.Equal(t, "C", updated.Language)
	})

	t.Run("no languages clears the main language", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		updated := &types.Repository{Language: "stale"}
		f.expectUpdate(updated)

		require.NoError(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID))
		require.Empty(t, updated.Language)
		require.Equal(t, [][]*types.RepoLangStat{nil}, f.langStore.inserted)
	})
}

func TestAnalyzeLanguages_Errors(t *testing.T) {
	boom := errors.New("boom")

	t.Run("repo lookup", func(t *testing.T) {
		f := newFixture(t)
		f.repoStore.On("Find", f.repo.ID).Return(nil, boom)
		require.ErrorIs(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID), boom)
	})

	t.Run("language stats", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, boom)
		require.ErrorIs(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID), boom)
		require.Empty(t, f.langStore.deleted)
	})

	t.Run("delete old languages", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		f.langStore.deleteErr = boom
		require.ErrorIs(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID), boom)
		require.Empty(t, f.langStore.inserted)
	})

	t.Run("insert new languages", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		f.langStore.insertErr = boom
		require.ErrorIs(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID), boom)
	})

	t.Run("transaction", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		f.analyzer.tx = fakeTx{err: boom}
		require.ErrorIs(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID), boom)
	})

	t.Run("repo update", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		f.repoStore.On("UpdateOptLock", f.repo, mock.Anything).Return(nil, boom)
		require.ErrorIs(t, f.analyzer.AnalyzeLanguages(t.Context(), f.repo.ID), boom)
	})
}

func TestEventHandlers(t *testing.T) {
	t.Run("default branch updated re-analyzes", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		f.expectUpdate(&types.Repository{})

		err := f.analyzer.defaultBranchUpdatedHandler(t.Context(),
			&events.Event[*repoevents.DefaultBranchUpdatedPayload]{
				Payload: &repoevents.DefaultBranchUpdatedPayload{Base: repoevents.Base{RepoID: f.repo.ID}},
			})
		require.NoError(t, err)
		require.Len(t, f.langStore.inserted, 1)
	})

	t.Run("default branch updated reports errors", func(t *testing.T) {
		f := newFixture(t)
		f.repoStore.On("Find", f.repo.ID).Return(nil, errors.New("boom"))

		err := f.analyzer.defaultBranchUpdatedHandler(t.Context(),
			&events.Event[*repoevents.DefaultBranchUpdatedPayload]{
				Payload: &repoevents.DefaultBranchUpdatedPayload{Base: repoevents.Base{RepoID: f.repo.ID}},
			})
		require.Error(t, err)
	})

	t.Run("imported repo created re-analyzes", func(t *testing.T) {
		f := newFixture(t)
		f.expectStats(git.GetRepoLanguageStatsOutput{}, nil)
		f.expectUpdate(&types.Repository{})

		err := f.analyzer.importedRepoCreatedHandler(t.Context(), &events.Event[*repoevents.CreatedPayload]{
			Payload: &repoevents.CreatedPayload{Base: repoevents.Base{RepoID: f.repo.ID}, ImportedFrom: "github"},
		})
		require.NoError(t, err)
	})

	t.Run("natively created repos are ignored", func(t *testing.T) {
		f := newFixture(t)
		err := f.analyzer.importedRepoCreatedHandler(t.Context(), &events.Event[*repoevents.CreatedPayload]{
			Payload: &repoevents.CreatedPayload{Base: repoevents.Base{RepoID: f.repo.ID}},
		})
		require.NoError(t, err)
	})

	t.Run("imported repo created reports errors", func(t *testing.T) {
		f := newFixture(t)
		f.repoStore.On("Find", f.repo.ID).Return(nil, errors.New("boom"))
		err := f.analyzer.importedRepoCreatedHandler(t.Context(), &events.Event[*repoevents.CreatedPayload]{
			Payload: &repoevents.CreatedPayload{Base: repoevents.Base{RepoID: f.repo.ID}, ImportedFrom: "github"},
		})
		require.Error(t, err)
	})
}
