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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCprMergeInput_Sanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      MergeInput
		wantErr bool
		want    enum.MergeMethod
	}{
		{name: "missing method", in: MergeInput{SourceSHA: cprSourceSHA}, wantErr: true},
		{name: "missing method on dry run", in: MergeInput{SourceSHA: cprSourceSHA, DryRun: true}},
		{name: "missing method on rules dry run", in: MergeInput{SourceSHA: cprSourceSHA, DryRunRules: true}},
		{name: "missing source sha", in: MergeInput{Method: enum.MergeMethodMerge}, wantErr: true},
		{name: "unknown method", in: MergeInput{Method: "bogus", SourceSHA: cprSourceSHA}, wantErr: true},
		{
			name: "squash with title", want: enum.MergeMethodSquash,
			in: MergeInput{Method: enum.MergeMethodSquash, SourceSHA: cprSourceSHA, Title: " t "},
		},
		{
			name: "rebase with title", wantErr: true,
			in: MergeInput{Method: enum.MergeMethodRebase, SourceSHA: cprSourceSHA, Title: "t"},
		},
		{
			name: "fast forward with message", wantErr: true,
			in: MergeInput{Method: enum.MergeMethodFastForward, SourceSHA: cprSourceSHA, Message: "m"},
		},
		{
			name: "rebase", want: enum.MergeMethodRebase,
			in: MergeInput{Method: enum.MergeMethodRebase, SourceSHA: cprSourceSHA, Title: "  "},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.in.sanitize()
			if tt.wantErr {
				require.Equal(t, http.StatusBadRequest, cprStatus(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, tt.in.Method)
		})
	}
}

func TestCprNeedsMergeCheck(t *testing.T) {
	t.Parallel()

	unchecked := &types.PullReq{
		MergeCheckStatus:  enum.MergeCheckStatusUnchecked,
		RebaseCheckStatus: enum.MergeCheckStatusMergeable,
	}
	checked := &types.PullReq{
		MergeCheckStatus:  enum.MergeCheckStatusMergeable,
		RebaseCheckStatus: enum.MergeCheckStatusUnchecked,
	}

	require.True(t, needsMergeCheck(unchecked, enum.MergeMethodMerge))
	require.True(t, needsMergeCheck(unchecked, enum.MergeMethodSquash))
	require.False(t, needsMergeCheck(unchecked, enum.MergeMethodRebase))
	require.False(t, needsMergeCheck(checked, enum.MergeMethodMerge))
	require.True(t, needsMergeCheck(checked, enum.MergeMethodRebase))
	require.True(t, needsMergeCheck(checked, enum.MergeMethodFastForward))
	require.True(t, needsMergeCheck(checked, "unknown"))
}

// cprMockMergeReads registers the git read calls made before a merge or merge dry run.
func cprMockMergeReads(e *cprEnv, pr *types.PullReq) {
	e.git.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
		return p.Type == gitenum.RefTypePullReqHead
	})).Return(git.GetRefResponse{SHA: sha.Must(pr.SourceSHA)}, nil).Maybe()
	e.git.On("GetBranch", mock.Anything, mock.Anything).
		Return(&git.GetBranchOutput{Branch: git.Branch{Name: "main", SHA: sha.Must(cprTargetSHA)}}, nil).Maybe()
	e.git.On("IsAncestor", mock.Anything, mock.Anything).
		Return(git.IsAncestorOutput{Ancestor: false}, nil).Maybe()
}

func cprMergeOutput(pr *types.PullReq) git.MergeOutput {
	return git.MergeOutput{
		BaseSHA:          sha.Must(cprTargetSHA),
		HeadSHA:          sha.Must(pr.SourceSHA),
		MergeBaseSHA:     sha.Must(cprBaseSHA),
		MergeSHA:         sha.Must("dddddddddddddddddddddddddddddddddddddddd"),
		CommitCount:      2,
		ChangedFileCount: 3,
		Additions:        4,
		Deletions:        5,
	}
}

func TestCprMerge_PreconditionErrors(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)
	in := func(pr *types.PullReq) *MergeInput {
		return &MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA}
	}

	t.Run("already merged", func(t *testing.T) {
		now := int64(5)
		pr := e.createPR(func(p *types.PullReq) { p.Merged = &now })
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number, in(pr))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("closed", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateClosed })
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number, in(pr))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("outdated source sha", func(t *testing.T) {
		pr := e.createPR()
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number,
			&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: cprTargetSHA})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("enqueued", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateMergeQueue })
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number, in(pr))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("draft", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.IsDraft = true })
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number, in(pr))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("invalid input", func(t *testing.T) {
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), 1, &MergeInput{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), 4242, in(&types.PullReq{SourceSHA: cprSourceSHA}))
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		pr := e.createPR()
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number, in(pr))
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("head ref mismatch", func(t *testing.T) {
		pr := e.createPR()
		e.git.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
			return p.Type == gitenum.RefTypePullReqHead
		})).Return(git.GetRefResponse{SHA: sha.Must(cprTargetSHA)}, nil).Once()
		_, _, err := e.ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number, in(pr))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

func TestCprMerge_DryRunRules(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)

	resp, violations, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
		&MergeInput{SourceSHA: pr.SourceSHA, DryRunRules: true})
	require.NoError(t, err)
	require.Nil(t, violations)
	require.True(t, resp.DryRunRules)
	require.Empty(t, resp.RuleViolations)
}

func TestCprMerge_DryRun(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	e.git.On("MergeVerify", mock.Anything, mock.Anything).Return(git.MergeVerifyOutput{
		MergeBaseSHA: sha.Must(cprBaseSHA), CommitCount: 1, ChangedFileCount: 2, Additions: 3, Deletions: 4,
	}, nil).Once()

	resp, _, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
		&MergeInput{SourceSHA: pr.SourceSHA, DryRun: true})
	require.NoError(t, err)
	require.True(t, resp.DryRun)
	require.True(t, resp.Mergeable)
	require.Empty(t, resp.ConflictFiles)

	stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, enum.MergeCheckStatusMergeable, stored.MergeCheckStatus)

	t.Run("stored result is reused", func(t *testing.T) {
		reused, _, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
			&MergeInput{SourceSHA: pr.SourceSHA, DryRun: true, Method: enum.MergeMethodMerge})
		require.NoError(t, err)
		require.True(t, reused.Mergeable)
	})

	t.Run("conflicts are reported", func(t *testing.T) {
		conflicted := e.createPR()
		cprMockMergeReads(e, conflicted)
		e.git.On("MergeVerify", mock.Anything, mock.Anything).Return(git.MergeVerifyOutput{
			MergeBaseSHA: sha.Must(cprBaseSHA), ConflictFiles: []string{"a.go"},
		}, nil).Once()
		got, _, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), conflicted.Number,
			&MergeInput{SourceSHA: conflicted.SourceSHA, DryRun: true})
		require.NoError(t, err)
		require.False(t, got.Mergeable)
		require.Equal(t, []string{"a.go"}, got.ConflictFiles)
	})

	t.Run("verify error", func(t *testing.T) {
		failing := e.createPR()
		cprMockMergeReads(e, failing)
		e.git.On("MergeVerify", mock.Anything, mock.Anything).
			Return(git.MergeVerifyOutput{}, errors.New("verify failed")).Once()
		_, _, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), failing.Number,
			&MergeInput{SourceSHA: failing.SourceSHA, DryRun: true})
		require.ErrorContains(t, err, "merge check")
	})
}

func TestCprMerge_Success(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	e.git.On("Merge", mock.Anything, mock.Anything).Return(cprMergeOutput(pr), nil).Once()
	e.git.On("UpdateRefs", mock.Anything, mock.Anything).Return(nil).Once()
	e.git.On("DeleteBranch", mock.Anything, mock.Anything).Return(nil).Once()

	repo, err := e.repoStore.Find(e.ctx, e.repo.ID)
	require.NoError(t, err)
	repo.NumOpenPulls = 1
	require.NoError(t, e.repoStore.Update(e.ctx, repo))

	resp, violations, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number, &MergeInput{
		Method:             enum.MergeMethodSquash,
		SourceSHA:          pr.SourceSHA,
		Title:              "squashed",
		Message:            "body",
		DeleteSourceBranch: true,
	})
	require.NoError(t, err)
	require.Nil(t, violations)
	require.Equal(t, cprMergeOutput(pr).MergeSHA.String(), resp.SHA)
	require.True(t, resp.BranchDeleted)

	stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, enum.PullReqStateMerged, stored.State)
	require.NotNil(t, stored.Merged)
	require.NotNil(t, stored.MergeMethod)
	require.Equal(t, enum.MergeMethodSquash, *stored.MergeMethod)

	repo, err = e.repoStore.Find(e.ctx, e.repo.ID)
	require.NoError(t, err)
	require.Equal(t, 0, repo.NumOpenPulls)
	require.Equal(t, 1, repo.NumMergedPulls)

	t.Run("merging again fails", func(t *testing.T) {
		_, _, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

func TestCprMerge_Conflicts(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	conflicted := cprMergeOutput(pr)
	conflicted.MergeSHA = sha.None
	conflicted.ConflictFiles = []string{"a.go", "b.go"}
	e.git.On("Merge", mock.Anything, mock.Anything).Return(conflicted, nil).Once()

	resp, violations, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
		&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
	require.NoError(t, err)
	require.Nil(t, resp)
	require.NotNil(t, violations)
	require.Equal(t, []string{"a.go", "b.go"}, violations.ConflictFiles)

	stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, enum.MergeCheckStatusConflict, stored.MergeCheckStatus)
	require.Equal(t, enum.PullReqStateOpen, stored.State)
}

func TestCprMerge_GitFailure(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	e.git.On("Merge", mock.Anything, mock.Anything).
		Return(git.MergeOutput{}, errors.New("merge exploded")).Once()

	_, _, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
		&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
	require.ErrorContains(t, err, "merge execution failed")
}

func TestCprMerge_RefUpdateFailure(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	e.git.On("Merge", mock.Anything, mock.Anything).Return(cprMergeOutput(pr), nil).Once()
	e.git.On("UpdateRefs", mock.Anything, mock.Anything).Return(errors.New("ref update failed")).Once()

	_, _, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
		&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
	require.ErrorContains(t, err, "after creating merge commit")

	// The failed ref update rolls back the database changes.
	stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, enum.PullReqStateOpen, stored.State)
}

func TestCprMerge_UnrelatedHistoriesDryRun(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	e.git.On("MergeVerify", mock.Anything, mock.Anything).
		Return(git.MergeVerifyOutput{}, errors.InvalidArgument("unrelated histories")).Once()

	_, _, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
		&MergeInput{SourceSHA: pr.SourceSHA, DryRun: true})
	require.Error(t, err)
}

func TestCprBackfillApprovalInfo(t *testing.T) {
	e := cprSetup(t)
	group := e.createUserGroup("cpr_approvers")

	approvals := []*types.DefaultReviewerApprovalsResponse{{
		PrincipalIDs: []int64{e.reviewer.ID, e.other.ID},
		UserGroupIDs: []int64{group.ID},
	}}
	require.NoError(t, e.ctrl.backfillApprovalInfo(e.ctx, approvals))
	require.Len(t, approvals[0].PrincipalInfos, 2)
	require.Len(t, approvals[0].UserGroupInfos, 1)
	require.Equal(t, group.Identifier, approvals[0].UserGroupInfos[0].Identifier)

	require.NoError(t, e.ctrl.backfillApprovalInfo(e.ctx, nil))
}
