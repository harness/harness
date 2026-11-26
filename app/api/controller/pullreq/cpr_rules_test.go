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
	"github.com/harness/gitness/app/services/checkreq"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	cprApprovalRule = `{"pullreq":{"approvals":{"require_minimum_count":1}}}`
	cprMergeQueue   = `{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["ci"]},` +
		`"group_size":1,"checks_concurrency":1,"max_check_duration_seconds":60}}}`
)

func TestCprMerge_BlockedByRule(t *testing.T) {
	e := cprSetup(t)
	e.createBranchRule("cpr-approvals", cprApprovalRule)
	pr := e.createPR()
	cprMockMergeReads(e, pr)

	t.Run("violations are returned", func(t *testing.T) {
		resp, violations, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
			&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
		require.NoError(t, err)
		require.Nil(t, resp)
		require.NotNil(t, violations)
		require.NotEmpty(t, violations.RuleViolations)
		require.Equal(t, "cpr-approvals", violations.RuleViolations[0].Rule.Identifier)
		require.NotEmpty(t, violations.Message)
	})

	t.Run("dry run rules lists the violations", func(t *testing.T) {
		resp, _, err := e.ctrl.Merge(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
			&MergeInput{SourceSHA: pr.SourceSHA, DryRunRules: true})
		require.NoError(t, err)
		require.True(t, resp.DryRunRules)
		require.NotEmpty(t, resp.RuleViolations)
	})
}

func TestCprMerge_BypassRule(t *testing.T) {
	e := cprSetup(t)
	e.createBranchRule("cpr-bypass",
		`{"bypass":{"user_ids":[1]},"pullreq":{"approvals":{"require_minimum_count":1},`+
			`"merge":{"require_bypass_message":true}}}`)
	pr := e.createPR()
	cprMockMergeReads(e, pr)

	t.Run("bypass message is required", func(t *testing.T) {
		_, _, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA, BypassRules: true})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("bypassed merge", func(t *testing.T) {
		e.git.On("Merge", mock.Anything, mock.Anything).Return(cprMergeOutput(pr), nil).Once()
		e.git.On("UpdateRefs", mock.Anything, mock.Anything).Return(nil).Once()

		resp, violations, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number, &MergeInput{
			Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA, BypassRules: true, BypassMessage: " urgent ",
		})
		require.NoError(t, err)
		require.Nil(t, violations)
		require.NotEmpty(t, resp.RuleViolations)

		stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Equal(t, enum.PullReqStateMerged, stored.State)
		require.NotNil(t, stored.MergeViolationsBypassed)
		require.True(t, *stored.MergeViolationsBypassed)
	})
}

func TestCprMerge_BranchInMergeQueue(t *testing.T) {
	e := cprSetup(t)
	queued := e.createPR(func(p *types.PullReq) { p.SourceBranch = "queued" })
	e.enqueue(queued)

	// A pull request stacked on top of the queued source branch can't be merged.
	pr := e.createPR(func(p *types.PullReq) { p.TargetBranch = "queued" })
	cprMockMergeReads(e, pr)

	resp, violations, err := e.ctrl.Merge(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
		&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
	require.NoError(t, err)
	require.Nil(t, resp)
	require.NotNil(t, violations)
	require.NotEmpty(t, violations.RuleViolations)
}

// enqueue puts the pull request into the merge queue of its target branch using the real services.
func (e *cprEnv) enqueue(pr *types.PullReq) {
	e.t.Helper()

	_, _, err := e.ctrl.mergeQueueService.Enqueue(
		e.ctx, pr, e.repo.Core(), e.author.ID, enum.MergeMethodMerge, "", "", false)
	require.NoError(e.t, err)
}

func TestCprListChecks(t *testing.T) {
	e := cprSetup(t)
	e.createBranchRule("cpr-checks", `{"pullreq":{"status_checks":{"require_identifiers":["unit","lint"]}}}`)
	pr := e.createPR()
	session := e.session(e.author)

	require.NoError(t, e.ctrl.checkStore.Upsert(e.ctx, &types.Check{
		CreatedBy:  e.author.ID,
		RepoID:     e.repo.ID,
		CommitSHA:  pr.SourceSHA,
		Identifier: "unit",
		Status:     enum.CheckStatusSuccess,
		Metadata:   []byte("{}"),
		Payload:    types.CheckPayload{Version: "1", Kind: enum.CheckPayloadKindEmpty, Data: []byte("{}")},
	}))
	require.NoError(t, e.ctrl.checkStore.Upsert(e.ctx, &types.Check{
		CreatedBy:  e.author.ID,
		RepoID:     e.repo.ID,
		CommitSHA:  pr.SourceSHA,
		Identifier: "extra",
		Status:     enum.CheckStatusFailure,
		Metadata:   []byte("{}"),
		Payload:    types.CheckPayload{Version: "1", Kind: enum.CheckPayloadKindEmpty, Data: []byte("{}")},
	}))

	e.checkReqs.reqs = checkreq.Requirements{
		Required:   []string{"external"},
		Bypassable: []string{"unit", "ext-bypassable"},
	}

	got, err := e.ctrl.ListChecks(e.ctx, session, e.repoRef(), pr.Number)
	require.NoError(t, err)
	require.Equal(t, pr.SourceSHA, got.CommitSHA)

	byID := map[string]types.PullReqCheck{}
	for _, c := range got.Checks {
		byID[c.Check.Identifier] = c
	}
	require.Len(t, byID, 5)

	require.True(t, byID["unit"].Required)
	require.Equal(t, enum.CheckStatusSuccess, byID["unit"].Check.Status)
	require.False(t, byID["extra"].Required)
	require.Equal(t, enum.CheckStatusFailure, byID["extra"].Check.Status)
	require.True(t, byID["lint"].Required)
	require.Equal(t, enum.CheckStatusPending, byID["lint"].Check.Status)
	require.True(t, byID["external"].Required)
	require.False(t, byID["external"].Bypassable)
	require.Equal(t, enum.CheckStatusPending, byID["external"].Check.Status)
	require.True(t, byID["ext-bypassable"].Bypassable)

	ids := make([]string, len(got.Checks))
	for i, c := range got.Checks {
		ids[i] = c.Check.Identifier
	}
	require.IsIncreasing(t, ids)

	t.Run("provider error", func(t *testing.T) {
		e.checkReqs.err = errors.New("provider down")
		_, err := e.ctrl.ListChecks(e.ctx, session, e.repoRef(), pr.Number)
		require.ErrorContains(t, err, "externally required")
		e.checkReqs.err = nil
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, err := e.ctrl.ListChecks(e.ctx, session, e.repoRef(), 4242)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.ListChecks(e.ctx, session, e.repoRef(), pr.Number)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprAutoMergeInput_Sanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      AutoMergeEnableInput
		wantErr bool
	}{
		{name: "missing method", in: AutoMergeEnableInput{}, wantErr: true},
		{name: "unknown method", in: AutoMergeEnableInput{Method: "bogus"}, wantErr: true},
		{name: "squash with title", in: AutoMergeEnableInput{Method: enum.MergeMethodSquash, Title: " t "}},
		{name: "rebase with message", in: AutoMergeEnableInput{Method: enum.MergeMethodRebase, Message: "m"}, wantErr: true},
		{name: "ff with title", in: AutoMergeEnableInput{Method: enum.MergeMethodFastForward, Title: "t"}, wantErr: true},
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
		})
	}
}

func TestCprVerifyIfAutoMergeable(t *testing.T) {
	t.Parallel()

	now := int64(1)
	tests := []struct {
		name    string
		pr      types.PullReq
		wantErr bool
	}{
		{name: "open", pr: types.PullReq{State: enum.PullReqStateOpen}},
		{name: "merged", pr: types.PullReq{State: enum.PullReqStateMerged, Merged: &now}, wantErr: true},
		{name: "closed", pr: types.PullReq{State: enum.PullReqStateClosed}, wantErr: true},
		{name: "draft", pr: types.PullReq{State: enum.PullReqStateOpen, IsDraft: true}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := verifyIfAutoMergeable(&tt.pr)
			if tt.wantErr {
				require.Equal(t, http.StatusBadRequest, cprStatus(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCprAutoMerge(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)
	enable := &AutoMergeEnableInput{Method: enum.MergeMethodSquash, Title: "t", Message: "m", DeleteSourceBranch: true}

	t.Run("setting must be enabled", func(t *testing.T) {
		pr := e.createPR()
		_, err := e.ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), pr.Number, enable)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	require.NoError(t, e.ctrl.settings.RepoSet(e.ctx, e.repo.ID, settings.KeyAutoMergeEnabled, true))
	e.createBranchRule("cpr-approvals", cprApprovalRule)

	t.Run("get without auto merge", func(t *testing.T) {
		pr := e.createPR()
		_, err := e.ctrl.AutoMergeGet(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("enable, get and disable", func(t *testing.T) {
		pr := e.createPR()
		cprMockMergeReads(e, pr)

		resp, err := e.ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), pr.Number, enable)
		require.NoError(t, err)
		require.Equal(t, enum.PullReqSubStateAutoMerge, resp.PullReqSubState)
		require.Equal(t, enum.MergeMethodSquash, resp.MergeMethod)
		require.True(t, resp.DeleteBranch)
		require.Nil(t, resp.MergeResponse)

		got, err := e.ctrl.AutoMergeGet(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Equal(t, "t", got.Title)
		require.Equal(t, "m", got.Message)
		require.Equal(t, e.author.ID, got.RequestedBy.ID)

		require.NoError(t, e.ctrl.AutoMergeDisable(e.ctx, session, e.repoRef(), pr.Number))
		_, err = e.ctrl.AutoMergeGet(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusNotFound, cprStatus(err))

		err = e.ctrl.AutoMergeDisable(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("disabled for closed pull request", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.State = enum.PullReqStateClosed })
		err := e.ctrl.AutoMergeDisable(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("enqueued pull request", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateMergeQueue })
		_, err := e.ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), pr.Number, enable)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("draft pull request", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.IsDraft = true })
		_, err := e.ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), pr.Number, enable)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, err := e.ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), 4242, enable)
		require.Error(t, err)
		_, err = e.ctrl.AutoMergeGet(e.ctx, session, e.repoRef(), 4242)
		require.Error(t, err)
	})

	t.Run("invalid input", func(t *testing.T) {
		_, err := e.ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), 1, &AutoMergeEnableInput{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.AutoMergeEnable(e.ctx, session, e.repoRef(), 1, enable)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		_, err = ctrl.AutoMergeGet(e.ctx, session, e.repoRef(), 1)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		require.ErrorIs(t, ctrl.AutoMergeDisable(e.ctx, session, e.repoRef(), 1), apiauth.ErrForbidden)
	})
}

func TestCprAutoMergeEnable_MergesImmediately(t *testing.T) {
	e := cprSetup(t)
	require.NoError(t, e.ctrl.settings.RepoSet(e.ctx, e.repo.ID, settings.KeyAutoMergeEnabled, true))
	pr := e.createPR()
	cprMockMergeReads(e, pr)
	e.git.On("Merge", mock.Anything, mock.Anything).Return(cprMergeOutput(pr), nil).Once()
	e.git.On("UpdateRefs", mock.Anything, mock.Anything).Return(nil).Once()
	e.git.On("DeleteBranch", mock.Anything, mock.Anything).Return(nil).Once()

	resp, err := e.ctrl.AutoMergeEnable(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
		&AutoMergeEnableInput{Method: enum.MergeMethodMerge, DeleteSourceBranch: true})
	require.NoError(t, err)
	require.NotNil(t, resp.MergeResponse)
	require.Equal(t, cprMergeOutput(pr).MergeSHA.String(), resp.MergeResponse.SHA)
	require.True(t, resp.MergeResponse.BranchDeleted)
	require.Equal(t, enum.PullReqStateMerged, resp.PullReqState)
}

func TestCprMergeQueueInput_Sanitize(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusBadRequest, cprStatus((&MergeQueueEnableInput{}).sanitize()))
	require.Equal(t, http.StatusBadRequest, cprStatus((&MergeQueueEnableInput{Method: "bogus"}).sanitize()))
	require.Equal(t, http.StatusBadRequest,
		cprStatus((&MergeQueueEnableInput{Method: enum.MergeMethodFastForward}).sanitize()))

	in := &MergeQueueEnableInput{Method: enum.MergeMethodSquash}
	require.NoError(t, in.sanitize())
	require.Equal(t, enum.MergeMethodSquash, in.Method)
}

func TestCprMergeQueue(t *testing.T) {
	e := cprSetup(t)
	session := e.session(e.author)
	enable := &MergeQueueEnableInput{Method: enum.MergeMethodMerge}

	t.Run("not configured", func(t *testing.T) {
		pr := e.createPR()
		cprMockMergeReads(e, pr)
		_, _, err := e.ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), pr.Number, enable)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	e.createBranchRule("cpr-mq", cprMergeQueue)

	t.Run("enable, get and prioritize", func(t *testing.T) {
		first := e.createPR()
		cprMockMergeReads(e, first)
		info, violations, err := e.ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), first.Number, enable)
		require.NoError(t, err)
		require.Nil(t, violations)
		require.NotNil(t, info)

		got, err := e.ctrl.MergeQueueGet(e.ctx, session, e.repoRef(), first.Number)
		require.NoError(t, err)
		require.NotNil(t, got)

		second := e.createPR()
		cprMockMergeReads(e, second)
		_, _, err = e.ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), second.Number, enable)
		require.NoError(t, err)

		err = e.ctrl.MergeQueuePrioritize(e.ctx, session, e.repoRef(), first.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err)) // already at the top of the queue

		require.NoError(t, e.ctrl.MergeQueuePrioritize(e.ctx, session, e.repoRef(), second.Number))
	})

	t.Run("enable twice", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateAutoMerge })
		_, _, err := e.ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), pr.Number, enable)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("get for pull request outside the queue", func(t *testing.T) {
		pr := e.createPR()
		_, err := e.ctrl.MergeQueueGet(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))

		pr = e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateMergeQueue })
		_, err = e.ctrl.MergeQueueGet(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("prioritize and remove outside the queue", func(t *testing.T) {
		pr := e.createPR()
		err := e.ctrl.MergeQueuePrioritize(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
		err = e.ctrl.MergeQueueRemove(e.ctx, session, e.repoRef(), pr.Number)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), 4242, enable)
		require.Error(t, err)
		_, err = e.ctrl.MergeQueueGet(e.ctx, session, e.repoRef(), 4242)
		require.Error(t, err)
		require.Error(t, e.ctrl.MergeQueuePrioritize(e.ctx, session, e.repoRef(), 4242))
		require.Error(t, e.ctrl.MergeQueueRemove(e.ctx, session, e.repoRef(), 4242))
	})

	t.Run("invalid input", func(t *testing.T) {
		_, _, err := e.ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), 1, &MergeQueueEnableInput{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.MergeQueueEnable(e.ctx, session, e.repoRef(), 1, enable)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		_, err = ctrl.MergeQueueGet(e.ctx, session, e.repoRef(), 1)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		require.ErrorIs(t, ctrl.MergeQueuePrioritize(e.ctx, session, e.repoRef(), 1), apiauth.ErrForbidden)
		require.ErrorIs(t, ctrl.MergeQueueRemove(e.ctx, session, e.repoRef(), 1), apiauth.ErrForbidden)
	})
}

func TestCprDeleteBranch_MergeQueueViolation(t *testing.T) {
	e := cprSetup(t)
	e.createBranchRule("cpr-mq", cprMergeQueue)
	queued := e.createPR()
	e.enqueue(queued)

	out, violations, err := e.ctrl.DeleteBranch(e.ctx, e.session(e.author), e.repoRef(), queued.Number, false, true)
	require.NoError(t, err)
	require.Nil(t, violations)
	require.NotEmpty(t, out.RuleViolations)

	e.git.On("GetBranch", mock.Anything, mock.Anything).
		Return(&git.GetBranchOutput{Branch: git.Branch{Name: queued.SourceBranch, SHA: sha.Must(queued.SourceSHA)}}, nil).
		Maybe()
	_, violations, err = e.ctrl.DeleteBranch(e.ctx, e.session(e.author), e.repoRef(), queued.Number, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, violations)
}
