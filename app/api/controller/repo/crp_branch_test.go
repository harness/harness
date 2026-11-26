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

package repo

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	crpDeleteForbidden    = `{"lifecycle":{"delete_forbidden":true}}`
	crpCreateForbidden    = `{"lifecycle":{"create_forbidden":true}}`
	crpCreateOwnerAllowed = `{"bypass":{"repo_owners":true},"lifecycle":{"create_forbidden":true}}`
	crpDeleteOwnerAllowed = `{"bypass":{"repo_owners":true},"lifecycle":{"delete_forbidden":true}}`
)

// crpCheckStore is a store.CheckStore with a preconfigured result summary.
type crpCheckStore struct {
	store.CheckStore
	summary map[sha.SHA]types.CheckCountSummary
	err     error
}

func (s *crpCheckStore) ResultSummary(
	context.Context, int64, []string, []enum.CheckPayloadKind,
) (map[sha.SHA]types.CheckCountSummary, error) {
	return s.summary, s.err
}

// crpPullReqStore is a store.PullReqStore with preconfigured open pull requests per branch.
type crpPullReqStore struct {
	store.PullReqStore
	byBranch map[string][]*types.PullReq
	err      error
	rootErr  error
}

func (s *crpPullReqStore) UpdateRootSpace(context.Context, []int64, int64, string) error {
	return s.rootErr
}

func (s *crpPullReqStore) ListOpenByBranchName(
	context.Context, int64, []string,
) (map[string][]*types.PullReq, error) {
	return s.byBranch, s.err
}

func crpBranch(name string, shaVal sha.SHA) git.Branch {
	return git.Branch{Name: name, SHA: shaVal}
}

func TestCrpMapToRPCSortOptions(t *testing.T) {
	branchTests := []struct {
		in   enum.BranchSortOption
		want git.BranchSortOption
	}{
		{enum.BranchSortOptionDate, git.BranchSortOptionDate},
		{enum.BranchSortOptionName, git.BranchSortOptionName},
		{enum.BranchSortOptionDefault, git.BranchSortOptionDefault},
		{enum.BranchSortOption(99), git.BranchSortOptionDefault},
	}
	for _, tt := range branchTests {
		assert.Equalf(t, tt.want, mapToRPCBranchSortOption(tt.in), "%v", tt.in)
	}

	tagTests := []struct {
		in   enum.TagSortOption
		want git.TagSortOption
	}{
		{enum.TagSortOptionDate, git.TagSortOptionDate},
		{enum.TagSortOptionName, git.TagSortOptionName},
		{enum.TagSortOptionDefault, git.TagSortOptionDefault},
		{enum.TagSortOption(99), git.TagSortOptionDefault},
	}
	for _, tt := range tagTests {
		assert.Equalf(t, tt.want, mapToRPCTagSortOption(tt.in), "%v", tt.in)
	}

	orderTests := []struct {
		in   enum.Order
		want git.SortOrder
	}{
		{enum.OrderAsc, git.SortOrderAsc},
		{enum.OrderDesc, git.SortOrderDesc},
		{enum.OrderDefault, git.SortOrderDefault},
		{enum.Order(99), git.SortOrderDefault},
	}
	for _, tt := range orderTests {
		assert.Equalf(t, tt.want, mapToRPCSortOrder(tt.in), "%v", tt.in)
	}
}

func TestCrpListBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("maps branches and marks the default one", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListBranches", mock.Anything, mock.MatchedBy(func(p *git.ListBranchesParams) bool {
			return p.Query == "fea" && p.Page == 2 && p.PageSize == 10 &&
				p.Sort == git.BranchSortOptionName && p.Order == git.SortOrderDesc && p.IncludeCommit
		})).Return(&git.ListBranchesOutput{
			Branches:   []git.Branch{crpBranch(crpDefaultBranch, crpTestSHA), crpBranch(crpFeatureBranch, crpTestSHAOther)},
			IsLastPage: true,
		}, nil)

		out, last, err := e.c.ListBranches(ctx, crpSession(), crpRepoRef, &types.BranchFilter{
			Query: "fea", Page: 2, Size: 10, Sort: enum.BranchSortOptionName, Order: enum.OrderDesc,
			IncludeCommit: true,
		})

		require.NoError(t, err)
		assert.True(t, last)
		require.Len(t, out, 2)
		assert.Equal(t, crpDefaultBranch, out[0].Name)
		assert.True(t, out[0].IsDefault)
		assert.False(t, out[1].IsDefault)
		assert.Nil(t, out[0].CheckSummary)
		assert.Nil(t, out[0].CommitDivergence)
	})

	t.Run("git failure is wrapped", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListBranches", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, _, err := e.c.ListBranches(ctx, crpSession(), crpRepoRef, &types.BranchFilter{})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("collects all optional metadata", func(t *testing.T) {
		e := crpNewEnv(t)
		e.c.checkStore = &crpCheckStore{summary: map[sha.SHA]types.CheckCountSummary{
			crpTestSHA: {Success: 3, Failure: 1},
		}}
		e.c.pullReqStore = &crpPullReqStore{byBranch: map[string][]*types.PullReq{
			crpDefaultBranch: {{Number: 5}},
		}}
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpDeleteForbidden)}
		e.git.On("ListBranches", mock.Anything, mock.Anything).Return(&git.ListBranchesOutput{
			Branches: []git.Branch{crpBranch(crpDefaultBranch, crpTestSHA)},
		}, nil)
		e.git.On("GetCommitDivergences", mock.Anything, mock.MatchedBy(func(p *git.GetCommitDivergencesParams) bool {
			return p.MaxCount == 100 && len(p.Requests) == 1 &&
				p.Requests[0].From == crpDefaultBranch && p.Requests[0].To == crpDefaultBranch
		})).Return(&git.GetCommitDivergencesOutput{
			Divergences: []api.CommitDivergence{{Ahead: 2, Behind: 4}},
		}, nil)

		out, _, err := e.c.ListBranches(ctx, crpSession(), crpRepoRef, &types.BranchFilter{
			BranchMetadataOptions: types.BranchMetadataOptions{
				IncludeChecks: true, IncludeRules: true, IncludePullReqs: true,
				IncludeMergeQueue: true, MaxDivergence: 100,
			},
		})

		require.NoError(t, err)
		require.Len(t, out, 1)
		require.NotNil(t, out[0].CheckSummary)
		assert.Equal(t, 3, out[0].CheckSummary.Success)
		require.Len(t, out[0].Rules, 1)
		assert.Equal(t, "rule1", out[0].Rules[0].Identifier)
		require.Len(t, out[0].PullRequests, 1)
		assert.EqualValues(t, 5, out[0].PullRequests[0].Number)
		require.NotNil(t, out[0].MergeQueue)
		assert.False(t, out[0].MergeQueue.Active)
		require.NotNil(t, out[0].CommitDivergence)
		assert.EqualValues(t, 2, out[0].CommitDivergence.Ahead)
		assert.EqualValues(t, 4, out[0].CommitDivergence.Behind)
	})

	metadataFailures := []struct {
		name  string
		setup func(e *crpEnv)
		opts  types.BranchMetadataOptions
	}{
		{
			name:  "check summary failure",
			setup: func(e *crpEnv) { e.c.checkStore = &crpCheckStore{err: errCrpBoom} },
			opts:  types.BranchMetadataOptions{IncludeChecks: true},
		},
		{
			name:  "rules failure",
			setup: func(e *crpEnv) { e.ruleStore.err = errCrpBoom },
			opts:  types.BranchMetadataOptions{IncludeRules: true},
		},
		{
			name:  "pull request failure",
			setup: func(e *crpEnv) { e.c.pullReqStore = &crpPullReqStore{err: errCrpBoom} },
			opts:  types.BranchMetadataOptions{IncludePullReqs: true},
		},
		{
			name:  "merge queue rules failure",
			setup: func(e *crpEnv) { e.ruleStore.err = errCrpBoom },
			opts:  types.BranchMetadataOptions{IncludeMergeQueue: true},
		},
		{
			name: "divergence failure",
			setup: func(e *crpEnv) {
				e.git.On("GetCommitDivergences", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
			},
			opts: types.BranchMetadataOptions{MaxDivergence: 5},
		},
	}
	for _, tt := range metadataFailures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			tt.setup(e)
			e.git.On("ListBranches", mock.Anything, mock.Anything).Return(&git.ListBranchesOutput{
				Branches: []git.Branch{crpBranch(crpDefaultBranch, crpTestSHA)},
			}, nil)

			_, _, err := e.c.ListBranches(ctx, crpSession(), crpRepoRef, &types.BranchFilter{
				BranchMetadataOptions: tt.opts,
			})

			require.ErrorIs(t, err, errCrpBoom)
		})
	}
}

func TestCrpGetBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the branch", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
			return p.BranchName == crpFeatureBranch
		})).Return(&git.GetBranchOutput{Branch: crpBranch(crpFeatureBranch, crpTestSHA)}, nil)

		out, err := e.c.GetBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, types.BranchMetadataOptions{})

		require.NoError(t, err)
		assert.Equal(t, crpFeatureBranch, out.Name)
		assert.Equal(t, crpTestSHA, out.SHA)
	})

	t.Run("verifies the signature of the head commit", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(&git.GetBranchOutput{
			Branch: git.Branch{Name: crpFeatureBranch, SHA: crpTestSHA, Commit: &git.Commit{SHA: crpTestSHA, Title: "t"}},
		}, nil)

		out, err := e.c.GetBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, types.BranchMetadataOptions{})

		require.NoError(t, err)
		require.NotNil(t, out.Commit)
		assert.Equal(t, "t", out.Commit.Title)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.GetBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, types.BranchMetadataOptions{})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("metadata failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.err = errCrpBoom
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(&git.GetBranchOutput{
			Branch: crpBranch(crpFeatureBranch, crpTestSHA),
		}, nil)

		_, err := e.c.GetBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch,
			types.BranchMetadataOptions{IncludeRules: true})

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpCreateBranch(t *testing.T) {
	ctx := context.Background()
	created := &git.CreateBranchOutput{Branch: crpBranch(crpFeatureBranch, crpTestSHA)}

	t.Run("target defaults to the default branch", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CreateBranch", mock.Anything, mock.MatchedBy(func(p *git.CreateBranchParams) bool {
			return p.BranchName == crpFeatureBranch && p.Target == crpDefaultBranch && p.RepoUID == crpGitUID
		})).Return(created, nil)

		out, violations, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef, &CreateBranchInput{Name: crpFeatureBranch})

		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.Equal(t, crpFeatureBranch, out.Name)
		assert.Empty(t, out.RuleViolations)
	})

	t.Run("explicit target is kept", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CreateBranch", mock.Anything, mock.MatchedBy(func(p *git.CreateBranchParams) bool {
			return p.Target == "abc123"
		})).Return(created, nil)

		_, _, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef,
			&CreateBranchInput{Name: crpFeatureBranch, Target: "abc123"})

		require.NoError(t, err)
	})

	t.Run("dry run reports violations without touching git", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpCreateForbidden)}

		out, violations, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef,
			&CreateBranchInput{Name: crpFeatureBranch, DryRunRules: true})

		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.True(t, out.DryRunRules)
		require.Len(t, out.RuleViolations, 1)
		e.git.AssertNotCalled(t, "CreateBranch", mock.Anything, mock.Anything)
	})

	t.Run("critical violation blocks creation", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoEdit: true}})
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpCreateForbidden)}

		_, violations, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef, &CreateBranchInput{Name: crpFeatureBranch})

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Equal(t, "lifecycle.create", violations[0].Violations[0].Code)
		e.git.AssertNotCalled(t, "CreateBranch", mock.Anything, mock.Anything)
	})

	t.Run("repo owner can bypass the violation", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpCreateOwnerAllowed)}
		e.git.On("CreateBranch", mock.Anything, mock.Anything).Return(created, nil)

		out, violations, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef,
			&CreateBranchInput{Name: crpFeatureBranch, BypassRules: true})

		require.NoError(t, err)
		assert.Empty(t, violations)
		require.Len(t, out.RuleViolations, 1)
		assert.True(t, out.RuleViolations[0].Bypassed)
	})

	t.Run("rule store failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.err = errCrpBoom

		_, _, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef, &CreateBranchInput{Name: crpFeatureBranch})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CreateBranch", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, _, err := e.c.CreateBranch(ctx, crpSession(), crpRepoRef, &CreateBranchInput{Name: crpFeatureBranch})

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpDeleteBranch_DefaultBranchCannotBeDeleted(t *testing.T) {
	e := crpNewEnv(t)

	_, _, err := e.c.DeleteBranch(context.Background(), crpSession(), crpRepoRef, crpDefaultBranch, false, false)

	require.ErrorIs(t, err, usererror.ErrDefaultBranchCantBeDeleted)
}

func TestCrpDeleteBranch_RuleStoreFailure(t *testing.T) {
	e := crpNewEnv(t)
	e.ruleStore.err = errCrpBoom

	_, _, err := e.c.DeleteBranch(context.Background(), crpSession(), crpRepoRef, crpFeatureBranch, false, false)

	require.ErrorIs(t, err, errCrpBoom)
}
