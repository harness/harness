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

package pullreq

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvaBoom = errors.New("boom")

type svaSpaceStore struct {
	store.SpaceStore

	descendants []types.SpaceParentData
	err         error
}

func (s *svaSpaceStore) GetDescendantsData(context.Context, int64) ([]types.SpaceParentData, error) {
	return s.descendants, s.err
}

type svaCheckStore struct {
	store.CheckStore

	summary map[sha.SHA]types.CheckCountSummary
	err     error
	calls   int
}

func (s *svaCheckStore) ResultSummary(
	_ context.Context,
	_ int64,
	_ []string,
	_ []enum.CheckPayloadKind,
) (map[sha.SHA]types.CheckCountSummary, error) {
	s.calls++
	return s.summary, s.err
}

type svaRuleStore struct {
	store.RuleStore

	rules []types.RuleInfoInternal
	err   error
}

func (s *svaRuleStore) ListAllRepoRules(
	context.Context, int64, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return s.rules, s.err
}

type svaLabelAssignmentStore struct {
	store.PullReqLabelAssignmentStore

	assigned map[int64][]*types.LabelPullReqAssignmentInfo
	err      error
}

func (s *svaLabelAssignmentStore) ListAssignedByPullreqIDs(
	context.Context, []int64,
) (map[int64][]*types.LabelPullReqAssignmentInfo, error) {
	return s.assigned, s.err
}

type svaGit struct {
	git.Interface

	diffStats func(params *git.DiffParams) (git.DiffStatsOutput, error)
}

func (g *svaGit) DiffStats(_ context.Context, params *git.DiffParams) (git.DiffStatsOutput, error) {
	return g.diffStats(params)
}

type svaListDeps struct {
	spaces  *svaSpaceStore
	prs     *svaPullReqStore
	checks  *svaCheckStore
	rules   *svaRuleStore
	labels  *svaLabelAssignmentStore
	gitSvc  *svaGit
	authz   *svaAuthorizer
	repos   map[int64]*types.RepositoryCore
	repoErr map[int64]error
}

func svaNewListDeps() *svaListDeps {
	return &svaListDeps{
		spaces: &svaSpaceStore{},
		prs:    &svaPullReqStore{},
		checks: &svaCheckStore{},
		rules:  &svaRuleStore{},
		labels: &svaLabelAssignmentStore{},
		gitSvc: &svaGit{diffStats: func(*git.DiffParams) (git.DiffStatsOutput, error) {
			return git.DiffStatsOutput{Commits: 2, FilesChanged: 3, Additions: 4, Deletions: 5}, nil
		}},
		authz: &svaAuthorizer{check: func(*types.Resource, *types.Scope) (bool, error) { return true, nil }},
		repos: map[int64]*types.RepositoryCore{
			1: {ID: 1, Path: "space/repo1", DefaultBranch: "main"},
			2: {ID: 2, Path: "space/repo2", DefaultBranch: "main"},
		},
		repoErr: map[int64]error{},
	}
}

func (d *svaListDeps) service() *ListService {
	finder := svaNewRepoFinder(func(_ context.Context, id int64) (*types.RepositoryCore, error) {
		if err := d.repoErr[id]; err != nil {
			return nil, err
		}
		repo, ok := d.repos[id]
		if !ok {
			return nil, gitness_store.ErrResourceNotFound
		}
		return repo, nil
	})

	return NewListService(
		nil,
		d.gitSvc,
		d.authz,
		d.spaces,
		d.prs,
		d.checks,
		finder,
		label.New(nil, nil, nil, nil, d.labels, nil, nil, refcache.SpaceFinder{}),
		protection.NewManager(d.rules),
	)
}

func TestSvaListServiceCountForSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		subspaces   bool
		descendants []types.SpaceParentData
		spaceErr    error
		countErr    error
		wantIDs     []int64
		wantCount   int64
		wantErr     bool
	}{
		{name: "single space", wantIDs: []int64{5}, wantCount: 7},
		{
			name:        "with subspaces",
			subspaces:   true,
			descendants: []types.SpaceParentData{{ID: 5}, {ID: 6}, {ID: 7}},
			wantIDs:     []int64{5, 6, 7},
			wantCount:   7,
		},
		{name: "descendants error", subspaces: true, spaceErr: errSvaBoom, wantErr: true},
		{name: "count error", countErr: errSvaBoom, wantIDs: []int64{5}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deps := svaNewListDeps()
			deps.spaces.descendants = test.descendants
			deps.spaces.err = test.spaceErr
			deps.prs.count = func(context.Context, *types.PullReqFilter) (int64, error) {
				return 7, test.countErr
			}

			filter := &types.PullReqFilter{}
			count, err := deps.service().CountForSpace(context.Background(), &types.SpaceCore{ID: 5}, test.subspaces, filter)
			if test.wantErr {
				require.ErrorIs(t, err, errSvaBoom)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.wantCount, count)
			require.Equal(t, test.wantIDs, filter.SpaceIDs)
		})
	}
}

func svaPR(id, repoID, updated int64) *types.PullReq {
	return &types.PullReq{
		ID:           id,
		Number:       id,
		TargetRepoID: repoID,
		TargetBranch: "main",
		SourceSHA:    "abc",
		Updated:      updated,
	}
}

func TestSvaListServiceListForSpace(t *testing.T) {
	t.Parallel()

	session := &auth.Session{Principal: types.Principal{ID: 10}}

	t.Run("filters forbidden and missing repos", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.repos[3] = &types.RepositoryCore{ID: 3, Path: "space/repo3"}
		deps.authz.check = func(res *types.Resource, _ *types.Scope) (bool, error) {
			return res.Identifier != "repo2", nil
		}
		deps.prs.stream = svaStreamOf([]*types.PullReq{
			svaPR(1, 1, 30), svaPR(2, 2, 20), svaPR(3, 3, 10), svaPR(4, 99, 5),
		}, nil)
		deps.labels.assigned = map[int64][]*types.LabelPullReqAssignmentInfo{1: {{LabelID: 77}}}

		filter := &types.PullReqFilter{Size: 10, Page: 4}
		filter.UpdatedGt = 5
		got, err := deps.service().ListForSpace(context.Background(), session, &types.SpaceCore{ID: 9}, false, filter)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, int64(1), got[0].PullRequest.ID)
		require.Equal(t, int64(3), got[1].PullRequest.ID)
		require.Equal(t, int64(1), got[0].Repository.ID)
		require.Len(t, got[0].PullRequest.Labels, 1)
		require.ElementsMatch(t, []int64{2, 99}, filter.RepoIDBlacklist)
		require.Equal(t, []int64{9}, filter.SpaceIDs)
		require.EqualValues(t, enum.PullReqSortUpdated, filter.Sort)
		require.EqualValues(t, enum.OrderDesc, filter.Order)
		require.Zero(t, filter.Page)
		require.Zero(t, filter.UpdatedGt)
	})

	t.Run("truncates to the requested size", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.prs.stream = svaStreamOf([]*types.PullReq{svaPR(1, 1, 30), svaPR(2, 1, 20)}, nil)

		got, err := deps.service().ListForSpace(
			context.Background(), session, &types.SpaceCore{ID: 9}, false, &types.PullReqFilter{Size: 1},
		)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, int64(1), got[0].PullRequest.ID)
	})

	t.Run("includes subspaces", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.spaces.descendants = []types.SpaceParentData{{ID: 9}, {ID: 10}}
		deps.prs.stream = svaStreamOf(nil, nil)

		filter := &types.PullReqFilter{Size: 5}
		got, err := deps.service().ListForSpace(context.Background(), session, &types.SpaceCore{ID: 9}, true, filter)
		require.NoError(t, err)
		require.Empty(t, got)
		require.Equal(t, []int64{9, 10}, filter.SpaceIDs)
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			subspaces bool
			setup     func(d *svaListDeps)
		}{
			{"descendants", true, func(d *svaListDeps) { d.spaces.err = errSvaBoom }},
			{"stream", false, func(d *svaListDeps) { d.prs.stream = svaStreamOf(nil, errSvaBoom) }},
			{"repo lookup", false, func(d *svaListDeps) {
				d.repoErr[1] = errSvaBoom
				d.prs.stream = svaStreamOf([]*types.PullReq{svaPR(1, 1, 1)}, nil)
			}},
			{"authorization", false, func(d *svaListDeps) {
				d.authz.check = func(*types.Resource, *types.Scope) (bool, error) { return false, errSvaBoom }
				d.prs.stream = svaStreamOf([]*types.PullReq{svaPR(1, 1, 1)}, nil)
			}},
			{"labels", false, func(d *svaListDeps) {
				d.labels.err = errSvaBoom
				d.prs.stream = svaStreamOf([]*types.PullReq{svaPR(1, 1, 1)}, nil)
			}},
			{"metadata", false, func(d *svaListDeps) {
				d.checks.err = errSvaBoom
				d.prs.stream = svaStreamOf([]*types.PullReq{svaPR(1, 1, 1)}, nil)
			}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				deps := svaNewListDeps()
				test.setup(deps)

				filter := &types.PullReqFilter{Size: 5}
				filter.IncludeChecks = test.name == "metadata"

				_, err := deps.service().ListForSpace(
					context.Background(), session, &types.SpaceCore{ID: 9}, test.subspaces, filter,
				)
				require.Error(t, err)
			})
		}
	})
}

func TestSvaListServiceBackfillMetadata(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, Path: "space/repo1", DefaultBranch: "main"}

	t.Run("no options clears diff stats", func(t *testing.T) {
		t.Parallel()

		pr := svaPR(1, 1, 1)
		pr.Stats.DiffStats = types.NewDiffStats(1, 2, 3, 4)

		err := svaNewListDeps().service().BackfillMetadataForPullReq(
			context.Background(), repo, pr, types.PullReqMetadataOptions{},
		)
		require.NoError(t, err)
		require.Equal(t, types.DiffStats{}, pr.Stats.DiffStats)
	})

	t.Run("source repo variants", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		sameRepo, otherRepo, missingRepo := int64(1), int64(2), int64(50)

		prNil := svaPR(1, 1, 1)
		prSame := svaPR(2, 1, 1)
		prSame.SourceRepoID = &sameRepo
		prOther := svaPR(3, 1, 1)
		prOther.SourceRepoID = &otherRepo
		prMissing := svaPR(4, 1, 1)
		prMissing.SourceRepoID = &missingRepo

		err := deps.service().BackfillMetadataForRepo(
			context.Background(), repo, []*types.PullReq{prNil, prSame, prOther, prMissing},
			types.PullReqMetadataOptions{},
		)
		require.NoError(t, err)
		require.Equal(t, "<deleted-repo>", prNil.SourceRepo.Identifier)
		require.Nil(t, prSame.SourceRepo)
		require.Equal(t, int64(2), prOther.SourceRepo.ID)
		require.Equal(t, "<deleted-repo>", prMissing.SourceRepo.Identifier)
	})

	t.Run("source repo lookup error", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.repoErr[2] = errSvaBoom
		other := int64(2)
		pr := svaPR(1, 1, 1)
		pr.SourceRepoID = &other

		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, pr, types.PullReqMetadataOptions{},
		)
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("checks", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.checks.summary = map[sha.SHA]types.CheckCountSummary{
			sha.Must("1111111111111111111111111111111111111111"): {Success: 3, Failure: 1},
		}
		pr := svaPR(1, 1, 1)
		pr.SourceSHA = "1111111111111111111111111111111111111111"
		other := svaPR(2, 1, 1)

		err := deps.service().BackfillMetadataForRepo(
			context.Background(), repo, []*types.PullReq{pr, other},
			types.PullReqMetadataOptions{IncludeChecks: true},
		)
		require.NoError(t, err)
		require.Equal(t, 1, deps.checks.calls)
		require.Equal(t, 3, pr.CheckSummary.Success)
		require.Equal(t, 1, pr.CheckSummary.Failure)
		require.NotNil(t, other.CheckSummary)
		require.Zero(t, other.CheckSummary.Success)
	})

	t.Run("checks error", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.checks.err = errSvaBoom

		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, svaPR(1, 1, 1), types.PullReqMetadataOptions{IncludeChecks: true},
		)
		require.Error(t, err)
	})

	t.Run("rules", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		pr := svaPR(1, 1, 1)

		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, pr, types.PullReqMetadataOptions{IncludeRules: true},
		)
		require.NoError(t, err)
		require.Empty(t, pr.Rules)
	})

	t.Run("rules error", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.rules.err = errSvaBoom

		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, svaPR(1, 1, 1), types.PullReqMetadataOptions{IncludeRules: true},
		)
		require.Error(t, err)
	})

	t.Run("git stats", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		var gotParams *git.DiffParams
		deps.gitSvc.diffStats = func(params *git.DiffParams) (git.DiffStatsOutput, error) {
			gotParams = params
			return git.DiffStatsOutput{Commits: 2, FilesChanged: 3, Additions: 4, Deletions: 5}, nil
		}

		pr := svaPR(1, 1, 1)
		pr.MergeBaseSHA = "base"
		pr.SourceSHA = "head"

		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, pr, types.PullReqMetadataOptions{IncludeGitStats: true},
		)
		require.NoError(t, err)
		require.Equal(t, types.NewDiffStats(2, 3, 4, 5), pr.Stats.DiffStats)
		require.Equal(t, "base", gotParams.BaseRef)
		require.Equal(t, "head", gotParams.HeadRef)
	})

	t.Run("git stats already known", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.gitSvc.diffStats = func(*git.DiffParams) (git.DiffStatsOutput, error) {
			return git.DiffStatsOutput{}, errSvaBoom
		}

		pr := svaPR(1, 1, 1)
		pr.Stats.DiffStats = types.NewDiffStats(9, 9, 9, 9)

		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, pr, types.PullReqMetadataOptions{IncludeGitStats: true},
		)
		require.NoError(t, err)
		require.Equal(t, types.NewDiffStats(9, 9, 9, 9), pr.Stats.DiffStats)
	})

	t.Run("git stats failures are not fatal", func(t *testing.T) {
		t.Parallel()

		deps := svaNewListDeps()
		deps.gitSvc.diffStats = func(*git.DiffParams) (git.DiffStatsOutput, error) {
			return git.DiffStatsOutput{}, errSvaBoom
		}

		pr := svaPR(1, 1, 1)
		err := deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, pr, types.PullReqMetadataOptions{IncludeGitStats: true},
		)
		require.NoError(t, err)
		require.Nil(t, pr.Stats.Commits)

		missing := svaPR(2, 77, 1)
		err = deps.service().BackfillMetadataForPullReq(
			context.Background(), repo, missing, types.PullReqMetadataOptions{IncludeGitStats: true},
		)
		require.NoError(t, err)
		require.Nil(t, missing.Stats.Commits)
	})
}
