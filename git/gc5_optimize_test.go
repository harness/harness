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

package git

import (
	"testing"
	"time"

	"github.com/harness/gitness/git/api"

	"github.com/stretchr/testify/require"
)

func TestGc5HeuristicalShouldRepackObjects(t *testing.T) {
	longAgo := time.Now().Add(-30 * 24 * time.Hour)

	tests := []struct {
		name         string
		info         api.RepositoryInfo
		wantRepack   bool
		wantStrategy RepackStrategy
	}{
		{name: "empty repository", info: api.RepositoryInfo{}},
		{
			name: "several packs and old full repack",
			info: api.RepositoryInfo{PackFiles: api.PackFilesInfo{Count: 3, LastFullRepack: longAgo}},

			wantRepack:   true,
			wantStrategy: RepackStrategyFullWithCruft,
		},
		{
			name: "several packs but recent full repack and no multi pack index",
			info: api.RepositoryInfo{PackFiles: api.PackFilesInfo{Count: 3, LastFullRepack: time.Now()}},

			wantRepack:   true,
			wantStrategy: RepackStrategyGeometric,
		},
		{
			name: "too many untracked packfiles",
			info: api.RepositoryInfo{PackFiles: api.PackFilesInfo{
				Count: 10, CruftCount: 9, LastFullRepack: time.Now(),
				MultiPackIndex: api.MultiPackIndexInfo{Exists: true, PackFileCount: 1},
			}},
			wantRepack:   true,
			wantStrategy: RepackStrategyGeometric,
		},
		{
			name: "many loose objects",
			info: api.RepositoryInfo{
				LooseObjects: api.LooseObjectsInfo{Count: LooseObjectLimit + 1},
				PackFiles: api.PackFilesInfo{
					Count: 1, LastFullRepack: time.Now(),
					MultiPackIndex: api.MultiPackIndexInfo{Exists: true, PackFileCount: 1},
				},
			},
			wantRepack:   true,
			wantStrategy: RepackStrategyIncrementalWithUnreachable,
		},
		{
			name: "tidy repository",
			info: api.RepositoryInfo{
				LooseObjects: api.LooseObjectsInfo{Count: 1},
				PackFiles: api.PackFilesInfo{
					Count: 1, LastFullRepack: time.Now(),
					MultiPackIndex: api.MultiPackIndexInfo{Exists: true, PackFileCount: 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needed, params := NewHeuristicalOptimizationStrategy(tt.info).ShouldRepackObjects(t.Context())
			require.Equal(t, tt.wantRepack, needed)
			if tt.wantRepack {
				require.Equal(t, tt.wantStrategy, params.Strategy)
				require.NoError(t, params.Validate())
			}
		})
	}
}

func TestGc5HeuristicalShouldPruneAndRepackRefs(t *testing.T) {
	ctx := t.Context()

	prune, params := NewHeuristicalOptimizationStrategy(api.RepositoryInfo{}).ShouldPruneObjects(ctx)
	require.False(t, prune)
	require.True(t, params.ExpireBefore.IsZero())

	prune, params = NewHeuristicalOptimizationStrategy(api.RepositoryInfo{
		LooseObjects: api.LooseObjectsInfo{StaleCount: LooseObjectLimit},
	}).ShouldPruneObjects(ctx)
	require.True(t, prune)
	require.False(t, params.ExpireBefore.IsZero())

	tests := []struct {
		name string
		refs api.ReferencesInfo
		want bool
	}{
		{"no loose refs", api.ReferencesInfo{}, false},
		{"few loose refs", api.ReferencesInfo{LooseReferenceCount: 5, PackedReferenceSize: 100}, false},
		{"many loose refs", api.ReferencesInfo{LooseReferenceCount: 100, PackedReferenceSize: 100}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHeuristicalOptimizationStrategy(api.RepositoryInfo{References: tt.refs})
			require.Equal(t, tt.want, s.ShouldRepackReferences(ctx))
		})
	}
}

func TestGc5HeuristicalShouldWriteCommitGraph(t *testing.T) {
	ctx := t.Context()

	write, _, err := NewHeuristicalOptimizationStrategy(api.RepositoryInfo{}).ShouldWriteCommitGraph(ctx)
	require.NoError(t, err)
	require.False(t, write)

	refs := api.ReferencesInfo{LooseReferenceCount: 1}

	// stale objects trigger pruning, which replaces the chain.
	write, params, err := NewHeuristicalOptimizationStrategy(api.RepositoryInfo{
		References:   refs,
		LooseObjects: api.LooseObjectsInfo{StaleCount: LooseObjectLimit},
	}).ShouldWriteCommitGraph(ctx)
	require.NoError(t, err)
	require.True(t, write)
	require.True(t, params.ReplaceChain)

	// full repack with cruft and expiry replaces the chain.
	write, params, err = NewHeuristicalOptimizationStrategy(api.RepositoryInfo{
		References: refs,
		PackFiles:  api.PackFilesInfo{Count: 3, LastFullRepack: time.Now().Add(-30 * 24 * time.Hour)},
	}).ShouldWriteCommitGraph(ctx)
	require.NoError(t, err)
	require.True(t, write)
	require.True(t, params.ReplaceChain)

	// geometric repack does not.
	write, params, err = NewHeuristicalOptimizationStrategy(api.RepositoryInfo{
		References: refs,
		PackFiles:  api.PackFilesInfo{Count: 1, LastFullRepack: time.Now()},
	}).ShouldWriteCommitGraph(ctx)
	require.NoError(t, err)
	require.True(t, write)
	require.False(t, params.ReplaceChain)

	// nothing to do.
	write, _, err = NewHeuristicalOptimizationStrategy(api.RepositoryInfo{References: refs}).ShouldWriteCommitGraph(ctx)
	require.NoError(t, err)
	require.False(t, write)
}

func TestGc5FullOptimizationStrategy(t *testing.T) {
	ctx := t.Context()
	s := NewFullOptimizationStrategy(api.RepositoryInfo{})

	repack, repackParams := s.ShouldRepackObjects(ctx)
	require.True(t, repack)
	require.Equal(t, RepackStrategyFullWithCruft, repackParams.Strategy)
	require.True(t, repackParams.WriteBitmap)
	require.False(t, repackParams.CruftExpireBefore.IsZero())

	write, graphParams, err := s.ShouldWriteCommitGraph(ctx)
	require.NoError(t, err)
	require.True(t, write)
	require.True(t, graphParams.ReplaceChain)

	prune, pruneParams := s.ShouldPruneObjects(ctx)
	require.True(t, prune)
	require.False(t, pruneParams.ExpireBefore.IsZero())

	require.True(t, s.ShouldRepackReferences(ctx))
}

func TestGc5OptimizeRepositoryStrategyValidate(t *testing.T) {
	require.NoError(t, OptimizeRepoStrategyGC.Validate())
	require.NoError(t, OptimizeRepoStrategyHeuristic.Validate())
	require.NoError(t, OptimizeRepoStrategyFull.Validate())
	require.Error(t, OptimizeRepoStrategy(42).Validate())

	require.Error(t, OptimizeRepositoryParams{}.Validate())
	require.Error(t, OptimizeRepositoryParams{ReadParams: ReadParams{RepoUID: "x"}, Strategy: 42}.Validate())
	require.NoError(t, OptimizeRepositoryParams{ReadParams: ReadParams{RepoUID: "x"}}.Validate())
}

func TestGc5OptimizeRepository(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	tests := []struct {
		name   string
		params OptimizeRepositoryParams
	}{
		{"gc", OptimizeRepositoryParams{ReadParams: f.read(), Strategy: OptimizeRepoStrategyGC}},
		{"gc with args", OptimizeRepositoryParams{
			ReadParams: f.read(), Strategy: OptimizeRepoStrategyGC, GCArgs: map[string]string{"aggressive": "true"},
		}},
		{"heuristic", OptimizeRepositoryParams{ReadParams: f.read(), Strategy: OptimizeRepoStrategyHeuristic}},
		{"full", OptimizeRepositoryParams{ReadParams: f.read(), Strategy: OptimizeRepoStrategyFull}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, f.svc.OptimizeRepository(ctx, tt.params))
			// repository is still readable afterwards.
			out, err := f.svc.GetDefaultBranch(ctx, &GetDefaultBranchParams{ReadParams: f.read()})
			require.NoError(t, err)
			require.Equal(t, "main", out.BranchName)
		})
	}

	t.Run("invalid params", func(t *testing.T) {
		require.Error(t, f.svc.OptimizeRepository(ctx, OptimizeRepositoryParams{}))
	})

	t.Run("unknown repository", func(t *testing.T) {
		require.Error(t, f.svc.OptimizeRepository(ctx, OptimizeRepositoryParams{
			ReadParams: ReadParams{RepoUID: "does-not-exist"}, Strategy: OptimizeRepoStrategyHeuristic,
		}))
	})
}
