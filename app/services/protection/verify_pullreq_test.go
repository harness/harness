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

	"github.com/harness/gitness/app/services/codeowners"
	gitrpcenum "github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// nolint:gocognit // it's a unit test
func TestDefPullReq_MergeVerify(t *testing.T) {
	tests := []struct {
		name      string
		def       DefPullReq
		in        MergeVerifyInput
		expCodes  []string
		expParams [][]any
		expOut    MergeVerifyOutput
	}{
		{
			name: "empty",
		},
		{
			name: codePullReqApprovalReqMinCount + "-fail",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 1}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc"},
				},
			},
			expCodes:  []string{codePullReqApprovalReqMinCount},
			expParams: [][]any{{0, 1}},
		},
		{
			name: codePullReqApprovalReqMinCount + "-success",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
				},
			},
		},
		{
			name: codePullReqApprovalReqLatestCommit + "-fail",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2, RequireLatestCommit: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abd"},
				},
			},
			expCodes:  []string{codePullReqApprovalReqMinCount},
			expParams: [][]any{{1, 2}},
		},
		{
			name: codePullReqApprovalReqLatestCommit + "-success",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2, RequireLatestCommit: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionPending, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc"},
				},
			},
		},
		{
			name: codePullReqApprovalReqCodeOwnersNoApproval + "-fail",
			def:  DefPullReq{Approvals: DefApprovals{RequireCodeOwners: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				CodeOwners: &codeowners.Evaluation{
					EvaluationEntries: []codeowners.EvaluationEntry{
						{
							Pattern: "app",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionPending, ReviewSHA: "abc"},
							},
						},
						{
							Pattern: "doc",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
						{
							Pattern:          "data",
							OwnerEvaluations: []codeowners.OwnerEvaluation{},
						},
					},
					FileSha: "xyz",
				},
			},
			expCodes: []string{
				codePullReqApprovalReqCodeOwnersNoApproval,
				codePullReqApprovalReqCodeOwnersNoApproval,
			},
			expParams: [][]any{{"app"}, {"data"}},
		},
		{
			name: codePullReqApprovalReqCodeOwnersNoApproval + "-success",
			def:  DefPullReq{Approvals: DefApprovals{RequireCodeOwners: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				CodeOwners: &codeowners.Evaluation{
					EvaluationEntries: []codeowners.EvaluationEntry{
						{
							Pattern: "app",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
						{
							Pattern: "doc",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
					},
					FileSha: "xyz",
				},
			},
		},
		{
			name: codePullReqApprovalReqCodeOwnersChangeRequested + "-fail",
			def:  DefPullReq{Approvals: DefApprovals{RequireCodeOwners: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				CodeOwners: &codeowners.Evaluation{
					EvaluationEntries: []codeowners.EvaluationEntry{
						{
							Pattern: "app",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
								{ReviewDecision: enum.PullReqReviewDecisionChangeReq, ReviewSHA: "abc"},
								{ReviewDecision: enum.PullReqReviewDecisionPending, ReviewSHA: "abc"},
							},
						},
						{
							Pattern: "data",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
					},
					FileSha: "xyz",
				},
			},
			expCodes:  []string{codePullReqApprovalReqCodeOwnersChangeRequested},
			expParams: [][]any{{"app"}},
		},
		{
			name: codePullReqApprovalReqCodeOwnersNoLatestApproval + "-fail",
			def:  DefPullReq{Approvals: DefApprovals{RequireCodeOwners: true, RequireLatestCommit: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				CodeOwners: &codeowners.Evaluation{
					EvaluationEntries: []codeowners.EvaluationEntry{
						{
							Pattern: "data",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "old"},
							},
						},
						{
							Pattern: "app",
							OwnerEvaluations: []codeowners.OwnerEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "old"},
							},
						},
					},
					FileSha: "xyz",
				},
			},
			expCodes:  []string{codePullReqApprovalReqCodeOwnersNoLatestApproval},
			expParams: [][]any{{"data"}},
		},
		{
			name: codePullReqCommentsReqResolveAll + "-fail",
			def:  DefPullReq{Comments: DefComments{RequireResolveAll: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 6},
			},
			expCodes:  []string{"pullreq.comments.require_resolve_all"},
			expParams: [][]any{{6}},
		},
		{
			name: codePullReqCommentsReqResolveAll + "-success",
			def:  DefPullReq{Comments: DefComments{RequireResolveAll: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0},
			},
		},
		{
			name: codePullReqStatusChecksReqUIDs + "-fail",
			def:  DefPullReq{StatusChecks: DefStatusChecks{RequireUIDs: []string{"check1"}}},
			in: MergeVerifyInput{
				CheckResults: []types.CheckResult{
					{UID: "check1", Status: enum.CheckStatusFailure},
					{UID: "check2", Status: enum.CheckStatusSuccess},
				},
			},
			expCodes:  []string{codePullReqStatusChecksReqUIDs},
			expParams: [][]any{nil},
		},
		{
			name: codePullReqStatusChecksReqUIDs + "-success",
			def:  DefPullReq{StatusChecks: DefStatusChecks{RequireUIDs: []string{"check1"}}},
			in: MergeVerifyInput{
				CheckResults: []types.CheckResult{
					{UID: "check1", Status: enum.CheckStatusSuccess},
					{UID: "check2", Status: enum.CheckStatusFailure},
				},
			},
		},
		{
			name: codePullReqMergeStrategiesAllowed + "-fail",
			def: DefPullReq{Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{
				enum.MergeMethod(gitrpcenum.MergeMethodRebase),
				enum.MergeMethod(gitrpcenum.MergeMethodSquash),
			}}},
			in: MergeVerifyInput{
				Method: enum.MergeMethod(gitrpcenum.MergeMethodMerge),
			},
			expCodes: []string{codePullReqMergeStrategiesAllowed},
			expParams: [][]any{{
				enum.MergeMethod(gitrpcenum.MergeMethodMerge),
				[]enum.MergeMethod{
					enum.MergeMethod(gitrpcenum.MergeMethodRebase),
					enum.MergeMethod(gitrpcenum.MergeMethodSquash),
				}},
			},
		},
		{
			name: codePullReqMergeStrategiesAllowed + "-success",
			def: DefPullReq{Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{
				enum.MergeMethod(gitrpcenum.MergeMethodRebase),
				enum.MergeMethod(gitrpcenum.MergeMethodSquash),
			}}},
			in: MergeVerifyInput{
				Method: enum.MergeMethod(gitrpcenum.MergeMethodSquash),
			},
		},
		{
			name:   codePullReqMergeDeleteBranch,
			def:    DefPullReq{Merge: DefMerge{DeleteBranch: true}},
			in:     MergeVerifyInput{},
			expOut: MergeVerifyOutput{DeleteSourceBranch: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.def.Sanitize(); err != nil {
				t.Errorf("def invalid: %s", err.Error())
				return
			}

			out, violations, err := test.def.MergeVerify(context.Background(), test.in)
			if err != nil {
				t.Errorf("got an error: %s", err.Error())
				return
			}

			if want, got := test.expOut, out; !reflect.DeepEqual(want, got) {
				t.Errorf("output mismatch: want=%+v got=%+v", want, got)
			}

			inspectBranchViolations(t, test.expCodes, test.expParams, violations)
		})
	}
}
