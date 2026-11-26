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
	"strings"
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

const (
	cprFeature   = "feature"
	cprSourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cprTargetSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	cprBaseSHA   = "cccccccccccccccccccccccccccccccccccccccc"
)

func cprRefMatcher(name string) interface{} {
	return mock.MatchedBy(func(p git.GetRefParams) bool { return p.Name == name })
}

// cprMockBranches registers GetRef answers for the cprFeature and "main" branches.
func cprMockBranches(e *cprEnv) {
	e.git.On("GetRef", mock.Anything, cprRefMatcher(cprFeature)).
		Return(git.GetRefResponse{SHA: sha.Must(cprSourceSHA)}, nil).Maybe()
	e.git.On("GetRef", mock.Anything, cprRefMatcher("main")).
		Return(git.GetRefResponse{SHA: sha.Must(cprTargetSHA)}, nil).Maybe()
}

func cprMockCreateGit(e *cprEnv) {
	cprMockBranches(e)
	e.git.On("MergeBase", mock.Anything, mock.Anything).
		Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Maybe()
	e.git.On("DiffStats", mock.Anything, mock.Anything).
		Return(git.DiffStatsOutput{Commits: 2, FilesChanged: 3, Additions: 10, Deletions: 4}, nil).Maybe()
	e.git.On("UpdateRef", mock.Anything, mock.Anything).Return(nil).Maybe()
}

func TestCprValidateHelpers(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateTitle("title"))
	require.Equal(t, http.StatusBadRequest, cprStatus(validateTitle("")))
	require.NoError(t, validateTitle(strings.Repeat("é", 256)))
	require.Equal(t, http.StatusBadRequest, cprStatus(validateTitle(strings.Repeat("é", 257))))

	require.NoError(t, validateDescription(strings.Repeat("a", 64<<10)))
	require.Equal(t, http.StatusBadRequest, cprStatus(validateDescription(strings.Repeat("a", 64<<10+1))))

	require.NoError(t, validateComment(strings.Repeat("a", 16<<10)))
	require.Equal(t, http.StatusBadRequest, cprStatus(validateComment(strings.Repeat("a", 16<<10+1))))
}

func TestCprCreateInput_Sanitize(t *testing.T) {
	t.Parallel()

	in := &CreateInput{Title: "  title ", Description: " body "}
	require.NoError(t, in.Sanitize())
	require.Equal(t, "title", in.Title)
	require.Equal(t, "body", in.Description)

	require.Error(t, (&CreateInput{Title: " "}).Sanitize())
	require.Error(t, (&CreateInput{Title: "t", Description: strings.Repeat("a", 64<<10+1)}).Sanitize())
}

func TestCprUpdateInput_Sanitize(t *testing.T) {
	t.Parallel()

	in := &UpdateInput{Title: "  title ", Description: " body "}
	require.NoError(t, in.Sanitize())
	require.Equal(t, "title", in.Title)
	require.Equal(t, "body", in.Description)

	require.Error(t, (&UpdateInput{Title: ""}).Sanitize())
	require.Error(t, (&UpdateInput{Title: "t", Description: strings.Repeat("a", 64<<10+1)}).Sanitize())
}

func TestCprStateInput_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   enum.PullReqState
		wantErr bool
	}{
		{"open", enum.PullReqStateOpen, false},
		{"closed", enum.PullReqStateClosed, false},
		{"merged", enum.PullReqStateMerged, true},
		{"unknown", "bogus", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&StateInput{State: tt.state}).Check()
			if tt.wantErr {
				require.Equal(t, http.StatusBadRequest, cprStatus(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCprCreate_Validation(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)

	t.Run("invalid title", func(t *testing.T) {
		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{Title: ""})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("same branch", func(t *testing.T) {
		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: "main", TargetBranch: "main",
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("denied on source repo", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceRepoRef: e.repoRef(), SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("empty repo ref", func(t *testing.T) {
		_, err := e.ctrl.Create(e.ctx, session, "", &CreateInput{Title: "t"})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("linked repo", func(t *testing.T) {
		e.repoIDCache.repoType = enum.RepoTypeLinked
		defer func() { e.repoIDCache.repoType = "" }()

		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.Equal(t, errors.StatusForbidden, errors.AsStatus(err))
	})

	t.Run("missing source branch", func(t *testing.T) {
		e.git.On("GetRef", mock.Anything, cprRefMatcher("missing")).
			Return(git.GetRefResponse{}, errors.NotFound("no branch")).Once()
		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: "missing", TargetBranch: "main",
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("empty source branch", func(t *testing.T) {
		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: "", TargetBranch: "main",
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("git error resolving source branch", func(t *testing.T) {
		e.git.On("GetRef", mock.Anything, cprRefMatcher("broken")).
			Return(git.GetRefResponse{}, errors.New("git down")).Once()
		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: "broken", TargetBranch: "main",
		})
		require.Error(t, err)
		require.Zero(t, cprStatus(err))
	})

	t.Run("missing target branch", func(t *testing.T) {
		cprMockBranches(e)
		e.git.On("GetRef", mock.Anything, cprRefMatcher("nomain")).
			Return(git.GetRefResponse{}, errors.NotFound("no branch")).Once()
		_, err := e.ctrl.Create(e.ctx, session, e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "nomain",
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

func TestCprCreate_Errors(t *testing.T) {
	t.Run("no new commits", func(t *testing.T) {
		e := cprSetup(t)
		cprMockBranches(e)
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprSourceSHA)}, nil).Once()

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("merge base error", func(t *testing.T) {
		e := cprSetup(t)
		cprMockBranches(e)
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{}, errors.New("merge base failed")).Once()

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.ErrorContains(t, err, "merge base")
	})

	t.Run("diff stats error", func(t *testing.T) {
		e := cprSetup(t)
		cprMockBranches(e)
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Once()
		e.git.On("DiffStats", mock.Anything, mock.Anything).
			Return(git.DiffStatsOutput{}, errors.New("stats failed")).Once()

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.ErrorContains(t, err, "diff stats")
	})

	t.Run("reviewer is the author", func(t *testing.T) {
		e := cprSetup(t)
		cprMockCreateGit(e)

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main", ReviewerIDs: []int64{e.author.ID},
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reviewer does not exist", func(t *testing.T) {
		e := cprSetup(t)
		cprMockCreateGit(e)

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main", ReviewerIDs: []int64{9999},
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request already exists", func(t *testing.T) {
		e := cprSetup(t)
		cprMockCreateGit(e)
		e.createPR(func(pr *types.PullReq) {
			pr.SourceBranch = cprFeature
			pr.TargetBranch = "main"
		})

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.Equal(t, http.StatusConflict, cprStatus(err))
	})

	t.Run("pull request head ref fails", func(t *testing.T) {
		e := cprSetup(t)
		cprMockBranches(e)
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Maybe()
		e.git.On("DiffStats", mock.Anything, mock.Anything).Return(git.DiffStatsOutput{}, nil).Maybe()
		e.git.On("UpdateRef", mock.Anything, mock.Anything).Return(errors.New("ref failed")).Once()

		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.ErrorContains(t, err, "PR head ref")

		// The transaction is rolled back: the repository counters weren't bumped.
		repo, err := e.repoStore.Find(e.ctx, e.repo.ID)
		require.NoError(t, err)
		require.EqualValues(t, 0, repo.PullReqSeq)
	})
}

func TestCprCreate_Success(t *testing.T) {
	e := cprSetup(t)
	cprMockCreateGit(e)

	pr, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
		Title:        " new feature ",
		Description:  "details",
		SourceBranch: cprFeature,
		TargetBranch: "main",
		IsDraft:      true,
		ReviewerIDs:  []int64{e.reviewer.ID},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, pr.Number)
	require.Equal(t, "new feature", pr.Title)
	require.True(t, pr.IsDraft)
	require.Equal(t, enum.PullReqStateOpen, pr.State)
	require.Equal(t, cprSourceSHA, pr.SourceSHA)
	require.Equal(t, cprBaseSHA, pr.MergeBaseSHA)
	require.NotNil(t, pr.Stats.Commits)
	require.EqualValues(t, 2, *pr.Stats.Commits)
	require.NotEmpty(t, e.streamer.published)

	reviewers, err := e.reviewerStore.List(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, reviewers, 1)
	require.Equal(t, e.reviewer.ID, reviewers[0].PrincipalID)

	repo, err := e.repoStore.Find(e.ctx, e.repo.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, repo.PullReqSeq)
	require.Equal(t, 1, repo.NumPulls)
	require.Equal(t, 1, repo.NumOpenPulls)

	e.git.AssertCalled(t, "UpdateRef", mock.Anything, mock.MatchedBy(func(p git.UpdateRefParams) bool {
		return p.Type == gitenum.RefTypePullReqHead && p.Name == "1"
	}))

	t.Run("duplicate is rejected afterwards", func(t *testing.T) {
		_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
			Title: "again", SourceBranch: cprFeature, TargetBranch: "main",
		})
		require.Equal(t, http.StatusConflict, cprStatus(err))
	})
}

func TestCprUpdate(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	t.Run("title and description", func(t *testing.T) {
		got, err := e.ctrl.Update(e.ctx, session, e.repoRef(), pr.Number, &UpdateInput{
			Title: "new title", Description: "new description",
		})
		require.NoError(t, err)
		require.Equal(t, "new title", got.Title)
		require.Equal(t, "new description", got.Description)

		acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
		require.NoError(t, err)
		var titleChanges int
		for _, a := range acts {
			if a.Type == enum.PullReqActivityTypeTitleChange {
				titleChanges++
			}
		}
		require.Equal(t, 1, titleChanges)
	})

	t.Run("only description writes no activity", func(t *testing.T) {
		got, err := e.ctrl.Update(e.ctx, session, e.repoRef(), pr.Number, &UpdateInput{
			Title: "new title", Description: "other description",
		})
		require.NoError(t, err)
		require.Equal(t, "other description", got.Description)
	})

	t.Run("no changes", func(t *testing.T) {
		got, err := e.ctrl.Update(e.ctx, session, e.repoRef(), pr.Number, &UpdateInput{
			Title: "new title", Description: "other description",
		})
		require.NoError(t, err)
		require.Equal(t, "new title", got.Title)
	})

	t.Run("invalid input", func(t *testing.T) {
		_, err := e.ctrl.Update(e.ctx, session, e.repoRef(), pr.Number, &UpdateInput{Title: ""})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		_, err := e.ctrl.Update(e.ctx, session, e.repoRef(), 4321, &UpdateInput{Title: "t"})
		require.Error(t, err)
	})

	t.Run("denied for push permission", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.Update(e.ctx, session, e.repoRef(), pr.Number, &UpdateInput{Title: "t"})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("purged source repo skips push check", func(t *testing.T) {
		forkless := e.createPR(func(p *types.PullReq) { p.SourceRepoID = nil })
		got, err := e.ctrl.Update(e.ctx, session, e.repoRef(), forkless.Number, &UpdateInput{Title: "t2"})
		require.NoError(t, err)
		require.Equal(t, "t2", got.Title)
	})

	t.Run("fork source repo is access checked", func(t *testing.T) {
		fork := e.createFork()
		forked := e.createPR(func(p *types.PullReq) { p.SourceRepoID = &fork.ID })

		got, err := e.ctrl.Update(e.ctx, session, e.repoRef(), forked.Number, &UpdateInput{Title: "t3"})
		require.NoError(t, err)
		require.Equal(t, "t3", got.Title)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.Update(e.ctx, session, e.repoRef(), forked.Number, &UpdateInput{Title: "t4"})
		require.Error(t, err)
	})
}

func TestCprState(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)
	cprMockCreateGit(e)

	t.Run("close then reopen", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) {
			p.SourceBranch = cprFeature
		})
		repo, err := e.repoStore.Find(e.ctx, e.repo.ID)
		require.NoError(t, err)
		repo.NumOpenPulls = 1
		require.NoError(t, e.repoStore.Update(e.ctx, repo))

		closed, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateClosed})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqStateClosed, closed.State)
		require.NotNil(t, closed.Closed)

		repo, err = e.repoStore.Find(e.ctx, e.repo.ID)
		require.NoError(t, err)
		require.Equal(t, 0, repo.NumOpenPulls)
		require.Equal(t, 1, repo.NumClosedPulls)

		// Closing an already closed pull request changes nothing.
		again, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateClosed})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqStateClosed, again.State)

		reopened, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateOpen})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqStateOpen, reopened.State)
		require.Nil(t, reopened.Closed)
		require.Equal(t, cprSourceSHA, reopened.SourceSHA)

		repo, err = e.repoStore.Find(e.ctx, e.repo.ID)
		require.NoError(t, err)
		require.Equal(t, 1, repo.NumOpenPulls)
		require.Equal(t, 0, repo.NumClosedPulls)
	})

	t.Run("draft toggle", func(t *testing.T) {
		pr := e.createPR()
		drafted, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number,
			&StateInput{State: enum.PullReqStateOpen, IsDraft: true})
		require.NoError(t, err)
		require.True(t, drafted.IsDraft)

		ready, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number,
			&StateInput{State: enum.PullReqStateOpen, IsDraft: false})
		require.NoError(t, err)
		require.False(t, ready.IsDraft)
	})

	t.Run("no change", func(t *testing.T) {
		pr := e.createPR()
		got, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateOpen})
		require.NoError(t, err)
		require.Equal(t, pr.Number, got.Number)
	})

	t.Run("merged pull request can't be modified", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateMerged })
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateClosed})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("enqueued pull request can't be modified", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateMergeQueue })
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateClosed})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("invalid state", func(t *testing.T) {
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), 1, &StateInput{State: "bogus"})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), 4242, &StateInput{State: enum.PullReqStateClosed})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		pr := e.createPR()
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateClosed})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("reopen with purged source repo", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) {
			p.State = enum.PullReqStateClosed
			p.SourceRepoID = nil
		})
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateOpen})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reopen with missing source branch", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) {
			p.State = enum.PullReqStateClosed
			p.SourceBranch = "gone"
		})
		e.git.On("GetRef", mock.Anything, cprRefMatcher("gone")).
			Return(git.GetRefResponse{}, errors.NotFound("no branch")).Once()
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateOpen})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reopen with duplicate open pull request", func(t *testing.T) {
		closed := e.createPR(func(p *types.PullReq) {
			p.State = enum.PullReqStateClosed
			p.SourceBranch = "dup"
		})
		e.createPR(func(p *types.PullReq) { p.SourceBranch = "dup" })
		e.git.On("GetRef", mock.Anything, cprRefMatcher("dup")).
			Return(git.GetRefResponse{SHA: sha.Must(cprSourceSHA)}, nil).Maybe()
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), closed.Number, &StateInput{State: enum.PullReqStateOpen})
		require.Equal(t, http.StatusConflict, cprStatus(err))
	})

	t.Run("reopen without new commits", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) {
			p.State = enum.PullReqStateClosed
			p.SourceBranch = "same"
		})
		e.git.On("GetRef", mock.Anything, cprRefMatcher("same")).
			Return(git.GetRefResponse{SHA: sha.Must(cprBaseSHA)}, nil).Maybe()
		e.git.On("MergeBase", mock.Anything, mock.MatchedBy(func(p git.MergeBaseParams) bool {
			return p.Ref1 == cprBaseSHA
		})).Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Once()
		_, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateOpen})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reopen fork pull request fetches objects", func(t *testing.T) {
		fork := e.createFork()
		pr := e.createPR(func(p *types.PullReq) {
			p.State = enum.PullReqStateClosed
			p.SourceRepoID = &fork.ID
			p.SourceBranch = cprFeature
		})
		e.git.On("FetchObjects", mock.Anything, mock.Anything).Return(git.FetchObjectsOutput{}, nil).Once()
		got, err := e.ctrl.State(e.ctx, session, e.repoRef(), pr.Number, &StateInput{State: enum.PullReqStateOpen})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqStateOpen, got.State)
	})
}

func TestCprFind(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()

	t.Run("authenticated sets watching", func(t *testing.T) {
		got, err := e.ctrl.Find(e.ctx, e.session(e.author), e.repoRef(), pr.Number, types.PullReqMetadataOptions{})
		require.NoError(t, err)
		require.Equal(t, pr.Number, got.Number)
		require.NotNil(t, got.Watching)
		require.False(t, *got.Watching)
	})

	t.Run("watcher", func(t *testing.T) {
		require.NoError(t, e.watcherStore.Create(e.ctx, pr.ID, e.reviewer.ID))
		got, err := e.ctrl.Find(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, types.PullReqMetadataOptions{})
		require.NoError(t, err)
		require.NotNil(t, got.Watching)
		require.True(t, *got.Watching)
	})

	t.Run("invalid number", func(t *testing.T) {
		_, err := e.ctrl.Find(e.ctx, e.session(e.author), e.repoRef(), 0, types.PullReqMetadataOptions{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("not found", func(t *testing.T) {
		_, err := e.ctrl.Find(e.ctx, e.session(e.author), e.repoRef(), 4242, types.PullReqMetadataOptions{})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.Find(e.ctx, e.session(e.author), e.repoRef(), pr.Number, types.PullReqMetadataOptions{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprFindByBranches(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR(func(p *types.PullReq) { p.SourceBranch = "topic" })
	session := e.session(e.author)
	opts := types.PullReqMetadataOptions{}

	t.Run("found", func(t *testing.T) {
		got, err := e.ctrl.FindByBranches(e.ctx, session, e.repoRef(), "topic", "main", opts)
		require.NoError(t, err)
		require.Equal(t, pr.Number, got.Number)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := e.ctrl.FindByBranches(e.ctx, session, e.repoRef(), "unknown", "main", opts)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("missing branches", func(t *testing.T) {
		_, err := e.ctrl.FindByBranches(e.ctx, session, e.repoRef(), "", "main", opts)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
		_, err = e.ctrl.FindByBranches(e.ctx, session, e.repoRef(), "topic", "", opts)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("upstream source is rejected", func(t *testing.T) {
		_, err := e.ctrl.FindByBranches(e.ctx, session, e.repoRef(), "upstream:topic", "main", opts)
		require.Error(t, err)
	})

	t.Run("upstream target without fork", func(t *testing.T) {
		_, err := e.ctrl.FindByBranches(e.ctx, session, e.repoRef(), "topic", "upstream:main", opts)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.FindByBranches(e.ctx, session, e.repoRef(), "topic", "main", opts)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprList(t *testing.T) {
	e := cprSetup(t)
	first := e.createPR()
	e.createPR()
	session := e.session(e.author)

	newFilter := func() *types.PullReqFilter {
		return &types.PullReqFilter{
			Page:  1,
			Size:  10,
			Sort:  enum.PullReqSortNumber,
			Order: enum.OrderAsc,
			States: []enum.PullReqState{
				enum.PullReqStateOpen,
			},
		}
	}

	t.Run("all", func(t *testing.T) {
		list, count, err := e.ctrl.List(e.ctx, session, e.repoRef(), newFilter())
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.EqualValues(t, 2, count)
		require.Equal(t, first.Number, list[0].Number)
	})

	t.Run("paginated uses count query", func(t *testing.T) {
		f := newFilter()
		f.Size = 1
		list, count, err := e.ctrl.List(e.ctx, session, e.repoRef(), f)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.EqualValues(t, 2, count)
	})

	t.Run("source repo same as target", func(t *testing.T) {
		f := newFilter()
		f.SourceRepoRef = e.repoRef()
		list, _, err := e.ctrl.List(e.ctx, session, e.repoRef(), f)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, e.repo.ID, f.SourceRepoID)
	})

	t.Run("different source repo", func(t *testing.T) {
		fork := e.createFork()
		f := newFilter()
		f.SourceRepoRef = "2"
		list, _, err := e.ctrl.List(e.ctx, session, e.repoRef(), f)
		require.NoError(t, err)
		require.Empty(t, list)
		require.Equal(t, fork.ID, f.SourceRepoID)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.List(e.ctx, session, e.repoRef(), newFilter())
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("unknown source repo", func(t *testing.T) {
		f := newFilter()
		f.SourceRepoRef = "999"
		_, _, err := e.ctrl.List(e.ctx, session, e.repoRef(), f)
		require.Error(t, err)
	})
}

func TestCprDiffAndCommits(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	t.Run("raw diff", func(t *testing.T) {
		e.git.On("RawDiff", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		var gotSource, gotBase string
		var sb strings.Builder
		err := e.ctrl.RawDiff(e.ctx, &sb, session, e.repoRef(), pr.Number, func(s, b string) {
			gotSource, gotBase = s, b
		})
		require.NoError(t, err)
		require.Equal(t, pr.SourceSHA, gotSource)
		require.Equal(t, pr.MergeBaseSHA, gotBase)
	})

	t.Run("raw diff errors", func(t *testing.T) {
		var sb strings.Builder
		require.Error(t, e.ctrl.RawDiff(e.ctx, &sb, session, e.repoRef(), 9999, nil))

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		require.ErrorIs(t, ctrl.RawDiff(e.ctx, &sb, session, e.repoRef(), pr.Number, nil), apiauth.ErrForbidden)
	})

	t.Run("diff", func(t *testing.T) {
		files := make(chan *git.FileDiff, 1)
		errs := make(chan error)
		files <- &git.FileDiff{Path: "a.go"}
		close(files)
		e.git.On("Diff", mock.Anything, mock.Anything, mock.Anything).
			Return((<-chan *git.FileDiff)(files), (<-chan error)(errs)).Once()

		var gotSource string
		stream, err := e.ctrl.Diff(e.ctx, session, e.repoRef(), pr.Number, func(s, _ string) {
			gotSource = s
		}, true, true)
		require.NoError(t, err)
		require.Equal(t, pr.SourceSHA, gotSource)

		diff, err := stream.Next()
		require.NoError(t, err)
		require.Equal(t, "a.go", diff.Path)
	})

	t.Run("diff errors", func(t *testing.T) {
		_, err := e.ctrl.Diff(e.ctx, session, e.repoRef(), 9999, nil, false, false)
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.Diff(e.ctx, session, e.repoRef(), pr.Number, nil, false, false)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("commits", func(t *testing.T) {
		e.git.On("ListCommits", mock.Anything, mock.Anything).
			Return(&git.ListCommitsOutput{}, nil).Once()
		commits, err := e.ctrl.Commits(e.ctx, session, e.repoRef(), pr.Number,
			&types.PaginationFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Empty(t, commits)
	})

	t.Run("commits git error", func(t *testing.T) {
		wantErr := errors.New("list failed")
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(nil, wantErr).Once()
		_, err := e.ctrl.Commits(e.ctx, session, e.repoRef(), pr.Number, &types.PaginationFilter{Page: 1, Limit: 10})
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("commits errors", func(t *testing.T) {
		_, err := e.ctrl.Commits(e.ctx, session, e.repoRef(), 9999, &types.PaginationFilter{})
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.Commits(e.ctx, session, e.repoRef(), pr.Number, &types.PaginationFilter{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}
