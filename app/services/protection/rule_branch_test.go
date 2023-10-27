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

package protection

import (
	"context"
	"reflect"
	"testing"

	gitrpcenum "github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestBranch_isBypass(t *testing.T) {
	user := &types.Principal{ID: 42}
	admin := &types.Principal{ID: 66, Admin: true}

	tests := []struct {
		name   string
		bypass DefBypass
		actor  *types.Principal
		owner  bool
		exp    bool
	}{
		{
			name:   "empty",
			bypass: DefBypass{UserIDs: nil, SpaceOwners: false},
			actor:  user,
			exp:    false,
		},
		{
			name:   "admin",
			bypass: DefBypass{UserIDs: nil, SpaceOwners: false},
			actor:  admin,
			exp:    true,
		},
		{
			name:   "space-owners-false",
			bypass: DefBypass{UserIDs: nil, SpaceOwners: false},
			actor:  user,
			owner:  true,
			exp:    false,
		},
		{
			name:   "space-owners-true",
			bypass: DefBypass{UserIDs: nil, SpaceOwners: true},
			actor:  user,
			owner:  true,
			exp:    true,
		},
		{
			name:   "selected-false",
			bypass: DefBypass{UserIDs: []int64{1, 66}, SpaceOwners: false},
			actor:  user,
			exp:    false,
		},
		{
			name:   "selected-true",
			bypass: DefBypass{UserIDs: []int64{1, 42, 66}, SpaceOwners: false},
			actor:  user,
			exp:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			branch := Branch{Bypass: test.bypass}
			if want, got := test.exp, branch.isBypassed(test.actor, test.owner); want != got {
				t.Errorf("want=%t got=%t", want, got)
			}
		})
	}
}

// nolint:gocognit // it's a unit test
func TestBranch_CanMerge(t *testing.T) {
	tests := []struct {
		name      string
		branch    Branch
		in        CanMergeInput
		expCodes  []string
		expParams [][]any
		expOut    CanMergeOutput
	}{
		{
			name: "empty",
		},
		{
			name: "pullreq.approvals.require_minimum_count-fail",
			branch: Branch{
				PullReq: DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 1}},
			},
			in: CanMergeInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc"},
				},
			},
			expCodes:  []string{"pullreq.approvals.require_minimum_count"},
			expParams: [][]any{{0, 1}},
		},
		{
			name: "pullreq.approvals.require_minimum_count-success",
			branch: Branch{
				PullReq: DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2}},
			},
			in: CanMergeInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
				},
			},
		},
		{
			name: "pullreq.approvals.require_latest_commit-fail",
			branch: Branch{
				PullReq: DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2, RequireLatestCommit: true}},
			},
			in: CanMergeInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abd"},
				},
			},
			expCodes:  []string{"pullreq.approvals.require_minimum_count"},
			expParams: [][]any{{1, 2}},
		},
		{
			name: "pullreq.approvals.require_latest_commit-success",
			branch: Branch{
				PullReq: DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2, RequireLatestCommit: true}},
			},
			in: CanMergeInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionPending, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
				},
			},
		},
		{
			name: "pullreq.comments.require_resolve_all-fail",
			branch: Branch{
				PullReq: DefPullReq{Comments: DefComments{RequireResolveAll: true}},
			},
			in: CanMergeInput{
				PullReq: &types.PullReq{UnresolvedCount: 6},
			},
			expCodes:  []string{"pullreq.comments.require_resolve_all"},
			expParams: [][]any{{6}},
		},
		{
			name: "pullreq.comments.require_resolve_all-success",
			branch: Branch{
				PullReq: DefPullReq{Comments: DefComments{RequireResolveAll: true}},
			},
			in: CanMergeInput{
				PullReq: &types.PullReq{UnresolvedCount: 0},
			},
		},
		{
			name: "pullreq.status_checks.required_uids-fail",
			branch: Branch{
				PullReq: DefPullReq{StatusChecks: DefStatusChecks{RequireUIDs: []string{"check1"}}},
			},
			in: CanMergeInput{
				CheckResults: []types.CheckResult{
					{UID: "check1", Status: enum.CheckStatusFailure},
					{UID: "check2", Status: enum.CheckStatusSuccess},
				},
			},
			expCodes:  []string{"pullreq.status_checks.required_uids"},
			expParams: [][]any{nil},
		},
		{
			name: "pullreq.status_checks.required_uids-success",
			branch: Branch{
				PullReq: DefPullReq{StatusChecks: DefStatusChecks{RequireUIDs: []string{"check1"}}},
			},
			in: CanMergeInput{
				CheckResults: []types.CheckResult{
					{UID: "check1", Status: enum.CheckStatusSuccess},
					{UID: "check2", Status: enum.CheckStatusFailure},
				},
			},
		},
		{
			name: "pullreq.merge.strategies_allowed-fail",
			branch: Branch{
				PullReq: DefPullReq{Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{
					enum.MergeMethod(gitrpcenum.MergeMethodRebase),
					enum.MergeMethod(gitrpcenum.MergeMethodSquash),
				}}},
			},
			in: CanMergeInput{
				Method: enum.MergeMethod(gitrpcenum.MergeMethodMerge),
			},
			expCodes: []string{"pullreq.merge.strategies_allowed"},
			expParams: [][]any{{
				enum.MergeMethod(gitrpcenum.MergeMethodMerge),
				[]enum.MergeMethod{
					enum.MergeMethod(gitrpcenum.MergeMethodRebase),
					enum.MergeMethod(gitrpcenum.MergeMethodSquash),
				}},
			},
		},
		{
			name: "pullreq.merge.strategies_allowed-success",
			branch: Branch{
				PullReq: DefPullReq{Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{
					enum.MergeMethod(gitrpcenum.MergeMethodRebase),
					enum.MergeMethod(gitrpcenum.MergeMethodSquash),
				}}},
			},
			in: CanMergeInput{
				Method: enum.MergeMethod(gitrpcenum.MergeMethodSquash),
			},
		},
		{
			name: "pullreq.merge.delete_branch",
			branch: Branch{
				PullReq: DefPullReq{Merge: DefMerge{DeleteBranch: true}},
			},
			in:     CanMergeInput{},
			expOut: CanMergeOutput{DeleteSourceBranch: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.branch.Sanitize(); err != nil {
				t.Errorf("branch input seems invalid: %s", err.Error())
				return
			}

			out, violations, err := test.branch.CanMerge(context.Background(), test.in)
			if err != nil {
				t.Errorf("got an error: %s", err.Error())
				return
			}

			if want, got := test.expOut, out; !reflect.DeepEqual(want, got) {
				t.Errorf("output mismatch: want=%+v got=%+v", want, got)
			}

			if len(test.expCodes) == 0 &&
				(len(violations) == 0 || len(violations) == 1 && len(violations[0].Violations) == 0) {
				// no violations expected and no violations received
				return
			}

			if len(violations) != 1 {
				t.Error("expected size of violation should always be one")
				return
			}

			if want, got := len(test.expCodes), len(violations[0].Violations); want != got {
				t.Errorf("violation count: want=%d got=%d", want, got)
				return
			}

			for i, violation := range violations[0].Violations {
				if want, got := test.expCodes[i], violation.Code; want != got {
					t.Errorf("violation %d code mismatch: want=%s got=%s", i, want, got)
				}
				if want, got := test.expParams[i], violation.Params; !reflect.DeepEqual(want, got) {
					t.Errorf("violation %d params mismatch: want=%v got=%v", i, want, got)
				}
			}
		})
	}
}

// nolint:gocognit // it's a unit test
func TestBranch_CanModifyRef(t *testing.T) {
	tests := []struct {
		name      string
		branch    Branch
		in        CanModifyRefInput
		expCodes  []string
		expParams [][]any
	}{
		{
			name: "empty",
		},
		{
			name:      "lifecycle.create-fail",
			branch:    Branch{Lifecycle: DefLifecycle{CreateForbidden: true}},
			in:        CanModifyRefInput{RefNames: []string{"a"}, RefAction: RefActionCreate, RefType: RefTypeBranch},
			expCodes:  []string{"lifecycle.create"},
			expParams: [][]any{{"a"}},
		},
		{
			name:      "lifecycle.delete-fail",
			branch:    Branch{Lifecycle: DefLifecycle{DeleteForbidden: true}},
			in:        CanModifyRefInput{RefNames: []string{"a"}, RefAction: RefActionDelete, RefType: RefTypeBranch},
			expCodes:  []string{"lifecycle.delete"},
			expParams: [][]any{{"a"}},
		},
		{
			name:      "lifecycle.update-fail",
			branch:    Branch{Lifecycle: DefLifecycle{UpdateForbidden: true}},
			in:        CanModifyRefInput{RefNames: []string{"a"}, RefAction: RefActionUpdate, RefType: RefTypeBranch},
			expCodes:  []string{"lifecycle.update"},
			expParams: [][]any{{"a"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.branch.Sanitize(); err != nil {
				t.Errorf("branch input seems invalid: %s", err.Error())
				return
			}

			violations, err := test.branch.CanModifyRef(context.Background(), test.in)
			if err != nil {
				t.Errorf("got an error: %s", err.Error())
				return
			}

			inspectBranchViolations(t, test.expCodes, test.expParams, violations)

			if len(test.expCodes) == 0 &&
				(len(violations) == 0 || len(violations) == 1 && len(violations[0].Violations) == 0) {
				// no violations expected and no violations received
				return
			}

			if len(violations) != 1 {
				t.Error("expected size of violation should always be one")
				return
			}

			if want, got := len(test.expCodes), len(violations[0].Violations); want != got {
				t.Errorf("violation count: want=%d got=%d", want, got)
				return
			}

			for i, violation := range violations[0].Violations {
				if want, got := test.expCodes[i], violation.Code; want != got {
					t.Errorf("violation %d code mismatch: want=%s got=%s", i, want, got)
				}
				if want, got := test.expParams[i], violation.Params; !reflect.DeepEqual(want, got) {
					t.Errorf("violation %d params mismatch: want=%v got=%v", i, want, got)
				}
			}
		})
	}
}

func inspectBranchViolations(t *testing.T,
	expCodes []string,
	expParams [][]any,
	violations []types.RuleViolations,
) {
	if len(expCodes) == 0 &&
		(len(violations) == 0 || len(violations) == 1 && len(violations[0].Violations) == 0) {
		// no violations expected and no violations received
		return
	}

	if len(violations) != 1 {
		t.Error("expected size of violation should always be one")
		return
	}

	if want, got := len(expCodes), len(violations[0].Violations); want != got {
		t.Errorf("violation count: want=%d got=%d", want, got)
		return
	}

	for i, violation := range violations[0].Violations {
		if want, got := expCodes[i], violation.Code; want != got {
			t.Errorf("violation %d code mismatch: want=%s got=%s", i, want, got)
		}
		if want, got := expParams[i], violation.Params; !reflect.DeepEqual(want, got) {
			t.Errorf("violation %d params mismatch: want=%v got=%v", i, want, got)
		}
	}
}
