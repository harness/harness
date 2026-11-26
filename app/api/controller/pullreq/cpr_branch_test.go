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
	"net/http"
	"testing"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCprDeleteBranch(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)

	t.Run("dry run reports no violations", func(t *testing.T) {
		pr := e.createPR()
		out, violations, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, true)
		require.NoError(t, err)
		require.Nil(t, violations)
		require.True(t, out.DryRunRules)
		require.Empty(t, out.RuleViolations)
	})

	t.Run("default branch can't be deleted", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SourceBranch = "main"; p.TargetBranch = "other" })
		_, _, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, false)
		require.ErrorIs(t, err, usererror.ErrDefaultBranchCantBeDeleted)
	})

	t.Run("success", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("GetBranch", mock.Anything, mock.Anything).
			Return(&git.GetBranchOutput{Branch: git.Branch{Name: pr.SourceBranch, SHA: sha.Must(pr.SourceSHA)}}, nil).
			Once()
		e.git.On("DeleteBranch", mock.Anything, mock.Anything).Return(nil).Once()

		out, violations, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, false)
		require.NoError(t, err)
		require.Nil(t, violations)
		require.False(t, out.DryRunRules)

		acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
		require.NoError(t, err)
		var found bool
		for _, a := range acts {
			found = found || a.Type == enum.PullReqActivityTypeBranchDelete
		}
		require.True(t, found)
	})

	t.Run("source branch moved", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("GetBranch", mock.Anything, mock.Anything).
			Return(&git.GetBranchOutput{Branch: git.Branch{Name: pr.SourceBranch, SHA: sha.Must(cprTargetSHA)}}, nil).
			Once()

		_, _, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, false)
		require.Equal(t, errors.StatusConflict, errors.AsStatus(err))
	})

	t.Run("git error when reading branch", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("GetBranch", mock.Anything, mock.Anything).
			Return((*git.GetBranchOutput)(nil), errors.New("boom")).Once()

		_, _, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, false)
		require.ErrorContains(t, err, "source branch")
	})

	t.Run("git error when deleting branch", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("GetBranch", mock.Anything, mock.Anything).
			Return(&git.GetBranchOutput{Branch: git.Branch{Name: pr.SourceBranch, SHA: sha.Must(pr.SourceSHA)}}, nil).
			Once()
		wantErr := errors.New("delete failed")
		e.git.On("DeleteBranch", mock.Anything, mock.Anything).Return(wantErr).Once()

		_, _, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, false)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.DeleteBranch(e.ctx, session, e.repoRef(), 4242, false, false)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		pr := e.createPR()
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.DeleteBranch(e.ctx, session, e.repoRef(), pr.Number, false, false)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprRestoreBranch(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)

	t.Run("open pull request branch exists", func(t *testing.T) {
		pr := e.createPR()
		_, _, err := e.ctrl.RestoreBranch(e.ctx, session, e.repoRef(), pr.Number, &RestoreBranchInput{})
		require.Equal(t, errors.StatusConflict, errors.AsStatus(err))
	})

	t.Run("dry run", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateClosed })
		out, violations, err := e.ctrl.RestoreBranch(
			e.ctx, session, e.repoRef(), pr.Number, &RestoreBranchInput{DryRunRules: true})
		require.NoError(t, err)
		require.Nil(t, violations)
		require.True(t, out.DryRunRules)
	})

	t.Run("success", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateClosed })
		e.git.On("CreateBranch", mock.Anything, mock.Anything).Return(&git.CreateBranchOutput{
			Branch: git.Branch{Name: pr.SourceBranch, SHA: sha.Must(pr.SourceSHA)},
		}, nil).Once()

		out, violations, err := e.ctrl.RestoreBranch(e.ctx, session, e.repoRef(), pr.Number, &RestoreBranchInput{})
		require.NoError(t, err)
		require.Nil(t, violations)
		require.Equal(t, pr.SourceBranch, out.Branch.Name)

		acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
		require.NoError(t, err)
		var found bool
		for _, a := range acts {
			found = found || a.Type == enum.PullReqActivityTypeBranchRestore
		}
		require.True(t, found)
	})

	t.Run("git error", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateClosed })
		wantErr := errors.New("create failed")
		e.git.On("CreateBranch", mock.Anything, mock.Anything).
			Return((*git.CreateBranchOutput)(nil), wantErr).Once()
		_, _, err := e.ctrl.RestoreBranch(e.ctx, session, e.repoRef(), pr.Number, &RestoreBranchInput{})
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.RestoreBranch(e.ctx, session, e.repoRef(), 4242, &RestoreBranchInput{})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.RestoreBranch(e.ctx, session, e.repoRef(), 1, &RestoreBranchInput{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprChangeTargetBranch(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)
	cprMockBranches(e)
	e.git.On("GetRef", mock.Anything, cprRefMatcher("develop")).
		Return(git.GetRefResponse{SHA: sha.Must(cprTargetSHA)}, nil).Maybe()
	e.git.On("GetRef", mock.Anything, cprRefMatcher("nonexistent")).
		Return(git.GetRefResponse{}, errors.NotFound("no branch")).Maybe()

	t.Run("success", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Once()
		e.git.On("DiffStats", mock.Anything, mock.Anything).
			Return(git.DiffStatsOutput{Commits: 1, FilesChanged: 2, Additions: 3, Deletions: 4}, nil).Once()

		got, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.NoError(t, err)
		require.Equal(t, "develop", got.TargetBranch)
		require.Equal(t, cprBaseSHA, got.MergeBaseSHA)
		require.Equal(t, enum.MergeCheckStatusUnchecked, got.MergeCheckStatus)

		acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
		require.NoError(t, err)
		var found bool
		for _, a := range acts {
			found = found || a.Type == enum.PullReqActivityTypeTargetBranchChange
		}
		require.True(t, found)
	})

	t.Run("same target is a no-op", func(t *testing.T) {
		pr := e.createPR()
		got, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "main"})
		require.NoError(t, err)
		require.Equal(t, "main", got.TargetBranch)
	})

	t.Run("target equals source", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SourceBranch = "develop" })
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.Equal(t, errors.StatusInvalidArgument, errors.AsStatus(err))
	})

	t.Run("no new commits", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(pr.SourceSHA)}, nil).Once()
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("merge base error", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{}, errors.New("no merge base")).Once()
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.ErrorContains(t, err, "merge base")
	})

	t.Run("diff stats error", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Once()
		e.git.On("DiffStats", mock.Anything, mock.Anything).
			Return(git.DiffStatsOutput{}, errors.New("stats failed")).Once()
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.ErrorContains(t, err, "diff stats")
	})

	t.Run("unknown new branch", func(t *testing.T) {
		pr := e.createPR()
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "nonexistent"})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("merged pull request", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateMerged })
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.Error(t, err)
	})

	t.Run("enqueued pull request", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateMergeQueue })
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), pr.Number,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, err := e.ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), 4242,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.ChangeTargetBranch(e.ctx, session, e.repoRef(), 1,
			&ChangeTargetBranchInput{BranchName: "develop"})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprPRBranchCandidates(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)

	now := time.Now().UnixMilli()
	for _, name := range []string{"main", "topic-a", "topic-b", "topic-c"} {
		require.NoError(t, e.ctrl.branchStore.Upsert(e.ctx, e.repo.ID, &types.BranchTable{
			Name: name, SHA: sha.Must(cprTargetSHA), CreatedBy: e.author.ID, Created: now,
			UpdatedBy: e.author.ID, Updated: now,
		}))
	}

	t.Run("default branch is excluded", func(t *testing.T) {
		got, err := e.ctrl.PRBranchCandidates(e.ctx, session, e.repoRef(), 10)
		require.NoError(t, err)
		for _, b := range got {
			require.NotEqual(t, "main", b.Name)
		}
		require.LessOrEqual(t, len(got), 3)
	})

	t.Run("limit", func(t *testing.T) {
		got, err := e.ctrl.PRBranchCandidates(e.ctx, session, e.repoRef(), 1)
		require.NoError(t, err)
		require.LessOrEqual(t, len(got), 1)
	})

	t.Run("missing default branch row", func(t *testing.T) {
		require.NoError(t, e.ctrl.branchStore.Delete(e.ctx, e.repo.ID, "main"))
		_, err := e.ctrl.PRBranchCandidates(e.ctx, session, e.repoRef(), 10)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.PRBranchCandidates(e.ctx, session, e.repoRef(), 10)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}
