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
	"net/http"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const crpForkID int64 = 8

// crpForkEnv adds a fork (repo crpForkID) of the default test repository.
func crpForkEnv(ctx context.Context, t *testing.T) *crpEnv {
	t.Helper()

	e := crpNewEnv(t)
	e.withMergeQueue(ctx, t, &crpMQEntryStore{})
	e.addRepo(&types.RepositoryCore{
		ID:            crpForkID,
		ParentID:      3,
		Identifier:    "fork",
		Path:          "space/fork",
		GitUID:        "forkuid",
		DefaultBranch: crpDefaultBranch,
		ForkID:        crpRepoID,
		State:         enum.RepoStateActive,
	})

	return e
}

func crpForkRef(t *testing.T, e *crpEnv, forkBranchSHA, upstreamSHA sha.SHA) {
	t.Helper()

	e.git.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
		return p.RepoUID == "forkuid"
	})).Return(git.GetRefResponse{SHA: forkBranchSHA}, nil)
	e.git.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
		return p.RepoUID == crpGitUID
	})).Return(git.GetRefResponse{SHA: upstreamSHA}, nil)
	e.git.On("FetchObjects", mock.Anything, mock.Anything).Return(git.FetchObjectsOutput{}, nil).Maybe()
}

func TestCrpForkSync(t *testing.T) {
	ctx := context.Background()
	forkRef := "8"
	in := func() *ForkSyncInput {
		return &ForkSyncInput{Branch: crpDefaultBranch, BranchCommitSHA: crpTestSHA}
	}

	t.Run("input validation", func(t *testing.T) {
		tests := []struct {
			name string
			in   *ForkSyncInput
		}{
			{name: "missing branch", in: &ForkSyncInput{BranchCommitSHA: crpTestSHA}},
			{name: "missing sha", in: &ForkSyncInput{Branch: crpDefaultBranch}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := crpForkEnv(ctx, t)

				_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, tt.in)

				require.Error(t, err)
				assert.True(t, errors.IsInvalidArgument(err))
			})
		}
	})

	t.Run("upstream branch defaults to the branch name", func(t *testing.T) {
		input := in()
		input.BranchUpstream = "  "
		require.NoError(t, input.sanitize())
		assert.Equal(t, crpDefaultBranch, input.BranchUpstream)
	})

	t.Run("fork branch lookup failure", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		e.git.On("GetRef", mock.Anything, mock.Anything).Return(git.GetRefResponse{}, errCrpBoom)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("stale branch commit", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		crpForkRef(t, e, crpTestSHAOther, crpTestSHAOther)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.Error(t, err)
		assert.True(t, errors.IsInvalidArgument(err))
		assert.ErrorContains(t, err, "isn't the latest commit")
	})

	t.Run("branch in merge queue", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{queued: map[string]struct{}{crpDefaultBranch: {}}})
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("merge queue failure", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{err: errCrpBoom})
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("repository is not a fork", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		e.git.On("GetRef", mock.Anything, mock.Anything).Return(git.GetRefResponse{SHA: crpTestSHA}, nil)

		_, _, err := e.c.ForkSync(ctx, crpSession(), crpRepoRef, in())

		require.Error(t, err)
		assert.ErrorContains(t, err, "not a fork")
	})

	t.Run("already up to date", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)
		e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{Ancestor: true}, nil)

		out, conflict, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.NoError(t, err)
		assert.Nil(t, conflict)
		assert.True(t, out.AlreadyAncestor)
	})

	t.Run("ancestor check failure", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)
		e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{}, errCrpBoom)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("merge base failure", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)
		e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{}, errCrpBoom)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.ErrorIs(t, err, errCrpBoom)
	})

	mergeMethods := []struct {
		name       string
		mergeBase  sha.SHA
		wantMethod gitenum.MergeMethod
		wantMerge  bool
	}{
		{name: "fast forward", mergeBase: crpTestSHA, wantMethod: gitenum.MergeMethodFastForward},
		{name: "merge commit", mergeBase: crpTestSHAOther, wantMethod: gitenum.MergeMethodMerge, wantMerge: true},
	}
	for _, tt := range mergeMethods {
		t.Run("syncs with "+tt.name, func(t *testing.T) {
			e := crpForkEnv(ctx, t)
			crpForkRef(t, e, crpTestSHA, crpTestSHAOther)
			e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{}, nil)
			e.git.On("MergeBase", mock.Anything, mock.Anything).
				Return(git.MergeBaseOutput{MergeBaseSHA: tt.mergeBase}, nil)
			newSHA := sha.Must("3333333333333333333333333333333333333333")
			e.git.On("Merge", mock.Anything, mock.MatchedBy(func(p *git.MergeParams) bool {
				return p.Method == tt.wantMethod && p.BaseSHA == crpTestSHA && p.HeadSHA == crpTestSHAOther &&
					(p.Message != "") == tt.wantMerge && len(p.Refs) == 1 && p.Refs[0].Name == "refs/heads/main"
			})).Return(git.MergeOutput{MergeSHA: newSHA}, nil)

			out, conflict, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

			require.NoError(t, err)
			assert.Nil(t, conflict)
			assert.Equal(t, newSHA, out.NewCommitSHA)
		})
	}

	t.Run("conflicts are reported", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)
		e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{MergeBaseSHA: crpTestSHAOther}, nil)
		e.git.On("Merge", mock.Anything, mock.Anything).Return(git.MergeOutput{ConflictFiles: []string{"x"}}, nil)

		out, conflict, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.NoError(t, err)
		assert.Nil(t, out)
		require.NotNil(t, conflict)
		assert.Equal(t, []string{"x"}, conflict.ConflictFiles)
	})

	t.Run("merge failure", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		crpForkRef(t, e, crpTestSHA, crpTestSHAOther)
		e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{MergeBaseSHA: crpTestSHA}, nil)
		e.git.On("Merge", mock.Anything, mock.Anything).Return(git.MergeOutput{}, errCrpBoom)

		_, _, err := e.c.ForkSync(ctx, crpSession(), forkRef, in())

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpUpstreamRevisionsInDotRanges(t *testing.T) {
	ctx := context.Background()

	t.Run("commits are listed from the upstream", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		e.git.On("ResolveRevision", mock.Anything, mock.MatchedBy(func(p git.ResolveRevisionParams) bool {
			return p.RepoUID == crpGitUID && p.Revision == crpDefaultBranch
		})).Return(git.ResolveRevisionOutput{SHA: crpTestSHAOther}, nil)
		e.git.On("FetchObjects", mock.Anything, mock.MatchedBy(func(p *git.FetchObjectsParams) bool {
			return p.Source == crpGitUID && len(p.ObjectSHAs) == 1 && p.ObjectSHAs[0] == crpTestSHAOther
		})).Return(git.FetchObjectsOutput{}, nil)
		e.git.On("ListCommits", mock.Anything, mock.MatchedBy(func(p *git.ListCommitsParams) bool {
			return p.RepoUID == "forkuid" && p.GitREF == crpFeatureBranch && p.After == crpTestSHAOther.String()
		})).Return(&git.ListCommitsOutput{}, nil)

		_, err := e.c.ListCommits(ctx, crpSession(), "8", crpFeatureBranch, &types.CommitFilter{After: "upstream:main"})

		require.NoError(t, err)
	})

	t.Run("upstream resolution failure", func(t *testing.T) {
		e := crpForkEnv(ctx, t)
		e.git.On("ResolveRevision", mock.Anything, mock.Anything).Return(git.ResolveRevisionOutput{}, errCrpBoom)

		_, err := e.c.ListCommits(ctx, crpSession(), "8", crpFeatureBranch, &types.CommitFilter{After: "upstream:main"})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("a non fork can't use upstream references", func(t *testing.T) {
		e := crpForkEnv(ctx, t)

		filter := &types.CommitFilter{After: "upstream:main"}

		_, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, crpFeatureBranch, filter)

		require.Error(t, err)
		assert.ErrorContains(t, err, "not a fork")
	})
}
