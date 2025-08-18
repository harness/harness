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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var (
	reviewer1 = types.PrincipalInfo{ID: 1, DisplayName: "Reviewer 1", UID: "reviewer-1"}
	reviewer2 = types.PrincipalInfo{ID: 2, DisplayName: "Reviewer 2", UID: "reviewer-2"}
	reviewer3 = types.PrincipalInfo{ID: 3, DisplayName: "Reviewer 3", UID: "reviewer-3"}
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
			name: "empty-with-merge-method",
			in: MergeVerifyInput{
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				DeleteSourceBranch: false,
				AllowedMethods:     enum.MergeMethods,
			},
		},
		{
			name: "empty-no-merge-method-specified",
			in:   MergeVerifyInput{},
			expOut: MergeVerifyOutput{
				AllowedMethods:     enum.MergeMethods,
				DeleteSourceBranch: false,
			},
		},
		{
			name: codePullReqApprovalReqMinCount + "-fail",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 1}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc", Reviewer: reviewer1},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqMinCount},
			expParams: [][]any{{0, 1}},
			expOut: MergeVerifyOutput{
				AllowedMethods:                enum.MergeMethods,
				MinimumRequiredApprovalsCount: 1,
			},
		},
		{
			name: codePullReqApprovalReqMinCount + "-success",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer1},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:                enum.MergeMethods,
				MinimumRequiredApprovalsCount: 2,
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
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqMinCountLatest},
			expParams: [][]any{{1, 2}},
			expOut: MergeVerifyOutput{
				AllowedMethods:                      enum.MergeMethods,
				MinimumRequiredApprovalsCountLatest: 2,
			},
		},
		{
			name: codePullReqApprovalReqLatestCommit + "-success",
			def:  DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 2, RequireLatestCommit: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionPending, SHA: "abc", Reviewer: reviewer1},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer3},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:                      enum.MergeMethods,
				MinimumRequiredApprovalsCountLatest: 2,
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-fail",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc", Reviewer: reviewer1},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqDefaultReviewerMinCount},
			expParams: [][]any{{0, 1}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer1.ID},
					CurrentCount:         0,
					MinimumRequiredCount: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-success",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer1},
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc", Reviewer: reviewer2},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer1.ID},
					CurrentCount:         1,
					MinimumRequiredCount: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-1-exact",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID}},
			},
			in: MergeVerifyInput{
				PullReq:   &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: nil,
				Method:    enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:           enum.MergeMethods,
				DefaultReviewerApprovals: nil,
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-1-more-fail",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID, reviewer2.ID}},
			},
			in: MergeVerifyInput{
				PullReq:   &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: nil,
				Method:    enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqDefaultReviewerMinCount},
			expParams: [][]any{{0, 1}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer2.ID},
					CurrentCount:         0,
					MinimumRequiredCount: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-1-more-success",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID, reviewer2.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer2.ID},
					CurrentCount:         1,
					MinimumRequiredCount: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-2-exact-fail",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 2},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID, reviewer2.ID}},
			},
			in: MergeVerifyInput{
				PullReq:   &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: []*types.PullReqReviewer{},
				Method:    enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqDefaultReviewerMinCount},
			expParams: [][]any{{0, 1}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer2.ID},
					CurrentCount:         0,
					MinimumRequiredCount: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-2-exact-success",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 2},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID, reviewer2.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer2.ID},
					CurrentCount:         1,
					MinimumRequiredCount: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-2-more-fail",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 2},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID, reviewer2.ID, reviewer3.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc", Reviewer: reviewer3},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqDefaultReviewerMinCount},
			expParams: [][]any{{1, 2}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer2.ID, reviewer3.ID},
					CurrentCount:         1,
					MinimumRequiredCount: 2,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCount + "-with-author-count-2-more-success",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 2},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID, reviewer2.ID, reviewer3.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc", Author: reviewer1},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer2},
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer3},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:         []int64{reviewer2.ID, reviewer3.ID},
					CurrentCount:         2,
					MinimumRequiredCount: 2,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCountLatest + "-fail",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1, RequireLatestCommit: true},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "def", Reviewer: reviewer1},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqDefaultReviewerMinCountLatest},
			expParams: [][]any{{0, 1}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:               []int64{reviewer1.ID},
					CurrentCount:               0,
					MinimumRequiredCountLatest: 1,
				}},
			},
		},
		{
			name: codePullReqApprovalReqDefaultReviewerMinCountLatest + "-success",
			def: DefPullReq{
				Approvals: DefApprovals{RequireMinimumDefaultReviewerCount: 1, RequireLatestCommit: true},
				Reviewers: DefReviewers{DefaultReviewerIDs: []int64{reviewer1.ID}},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0, SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "abc", Reviewer: reviewer1},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
				DefaultReviewerApprovals: []*types.DefaultReviewerApprovalsResponse{{
					PrincipalIDs:               []int64{reviewer1.ID},
					CurrentCount:               1,
					MinimumRequiredCountLatest: 1,
				}},
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
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionPending, ReviewSHA: "abc"},
							},
						},
						{
							Pattern: "doc",
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
						{
							Pattern:         "data",
							UserEvaluations: []codeowners.UserEvaluation{},
						},
					},
					FileSha: "xyz",
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes: []string{
				codePullReqApprovalReqCodeOwnersNoApproval,
				codePullReqApprovalReqCodeOwnersNoApproval,
			},
			expParams: [][]any{{"app"}, {"data"}},
			expOut: MergeVerifyOutput{
				AllowedMethods:             enum.MergeMethods,
				RequiresCodeOwnersApproval: true,
			},
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
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
						{
							Pattern: "doc",
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
					},
					FileSha: "xyz",
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:             enum.MergeMethods,
				RequiresCodeOwnersApproval: true,
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
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
								{ReviewDecision: enum.PullReqReviewDecisionChangeReq, ReviewSHA: "abc"},
								{ReviewDecision: enum.PullReqReviewDecisionPending, ReviewSHA: "abc"},
							},
						},
						{
							Pattern: "data",
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
							},
						},
					},
					FileSha: "xyz",
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqCodeOwnersChangeRequested},
			expParams: [][]any{{"app"}},
			expOut: MergeVerifyOutput{
				AllowedMethods:             enum.MergeMethods,
				RequiresCodeOwnersApproval: true,
			},
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
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "old"},
							},
						},
						{
							Pattern: "app",
							UserEvaluations: []codeowners.UserEvaluation{
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "abc"},
								{ReviewDecision: enum.PullReqReviewDecisionApproved, ReviewSHA: "old"},
							},
						},
					},
					FileSha: "xyz",
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqCodeOwnersNoLatestApproval},
			expParams: [][]any{{"data"}},
			expOut: MergeVerifyOutput{
				AllowedMethods:                   enum.MergeMethods,
				RequiresCodeOwnersApprovalLatest: true,
			},
		},
		{
			name: codePullReqCommentsReqResolveAll + "-fail",
			def:  DefPullReq{Comments: DefComments{RequireResolveAll: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 6},
				Method:  enum.MergeMethodMerge,
			},
			expCodes:  []string{"pullreq.comments.require_resolve_all"},
			expParams: [][]any{{6}},
			expOut: MergeVerifyOutput{
				AllowedMethods:            enum.MergeMethods,
				RequiresCommentResolution: true,
			},
		},
		{
			name: codePullReqCommentsReqResolveAll + "-success",
			def:  DefPullReq{Comments: DefComments{RequireResolveAll: true}},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{UnresolvedCount: 0},
				Method:  enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:            enum.MergeMethods,
				RequiresCommentResolution: true,
			},
		},
		{
			name: codePullReqStatusChecksReqIdentifiers + "-fail",
			def:  DefPullReq{StatusChecks: DefStatusChecks{RequireIdentifiers: []string{"check1"}}},
			in: MergeVerifyInput{
				CheckResults: []types.CheckResult{
					{Identifier: "check1", Status: enum.CheckStatusFailure},
					{Identifier: "check2", Status: enum.CheckStatusSuccess},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqStatusChecksReqIdentifiers},
			expParams: [][]any{{"check1"}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
			},
		},
		{
			name: codePullReqStatusChecksReqIdentifiers + "-missing",
			def:  DefPullReq{StatusChecks: DefStatusChecks{RequireIdentifiers: []string{"check1"}}},
			in: MergeVerifyInput{
				CheckResults: []types.CheckResult{
					{Identifier: "check2", Status: enum.CheckStatusSuccess},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqStatusChecksReqIdentifiers},
			expParams: [][]any{{"check1"}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
			},
		},
		{
			name: codePullReqStatusChecksReqIdentifiers + "-success",
			def:  DefPullReq{StatusChecks: DefStatusChecks{RequireIdentifiers: []string{"check1"}}},
			in: MergeVerifyInput{
				CheckResults: []types.CheckResult{
					{Identifier: "check1", Status: enum.CheckStatusSuccess},
					{Identifier: "check2", Status: enum.CheckStatusFailure},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
			},
		},
		{
			name: codePullReqMergeStrategiesAllowed + "-fail",
			def: DefPullReq{Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{
				enum.MergeMethodRebase,
				enum.MergeMethodSquash,
			}}},
			in: MergeVerifyInput{
				Method: enum.MergeMethodMerge,
			},
			expCodes: []string{codePullReqMergeStrategiesAllowed},
			expParams: [][]any{{
				enum.MergeMethodMerge,
				[]enum.MergeMethod{
					enum.MergeMethodRebase,
					enum.MergeMethodSquash,
				}},
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: []enum.MergeMethod{enum.MergeMethodRebase, enum.MergeMethodSquash},
			},
		},
		{
			name: codePullReqMergeStrategiesAllowed + "-success",
			def: DefPullReq{Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{
				enum.MergeMethodRebase,
				enum.MergeMethodSquash,
			}}},
			in: MergeVerifyInput{
				Method: enum.MergeMethodSquash,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: []enum.MergeMethod{enum.MergeMethodRebase, enum.MergeMethodSquash},
			},
		},
		{
			name: codePullReqMergeDeleteBranch,
			def:  DefPullReq{Merge: DefMerge{DeleteBranch: true}},
			in: MergeVerifyInput{
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:     enum.MergeMethods,
				DeleteSourceBranch: true,
			},
		},
		{
			name: codePullReqApprovalReqChangeRequested + "-true",
			def: DefPullReq{
				Approvals: DefApprovals{RequireNoChangeRequest: true},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{SourceSHA: "abc"},
				Method:  enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods:           enum.MergeMethods,
				RequiresNoChangeRequests: true,
			},
		},
		{
			name: codePullReqApprovalReqChangeRequested + "-false",
			def: DefPullReq{
				Approvals: DefApprovals{RequireNoChangeRequest: false},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{ReviewDecision: enum.PullReqReviewDecisionChangeReq, SHA: "abc", Reviewer: reviewer1},
				},
				Method: enum.MergeMethodMerge,
			},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
			},
		},
		{
			name: codePullReqApprovalReqChangeRequested + "-sameSHA",
			def: DefPullReq{
				Approvals: DefApprovals{RequireNoChangeRequest: true},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{
						ReviewDecision: enum.PullReqReviewDecisionChangeReq,
						Reviewer:       reviewer1,
						SHA:            "abc",
					},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqChangeRequested},
			expParams: [][]any{{reviewer1.DisplayName}},
			expOut: MergeVerifyOutput{
				AllowedMethods:           enum.MergeMethods,
				RequiresNoChangeRequests: true,
			},
		},
		{
			name: codePullReqApprovalReqChangeRequested + "-diffSHA",
			def: DefPullReq{
				Approvals: DefApprovals{RequireNoChangeRequest: true},
			},
			in: MergeVerifyInput{
				PullReq: &types.PullReq{SourceSHA: "abc"},
				Reviewers: []*types.PullReqReviewer{
					{
						ReviewDecision: enum.PullReqReviewDecisionChangeReq,
						Reviewer:       reviewer1,
						SHA:            "def",
					},
				},
				Method: enum.MergeMethodMerge,
			},
			expCodes:  []string{codePullReqApprovalReqChangeRequestedOldSHA},
			expParams: [][]any{{reviewer1.DisplayName}},
			expOut: MergeVerifyOutput{
				AllowedMethods:           enum.MergeMethods,
				RequiresNoChangeRequests: true,
			},
		},
		{
			name: codePullReqMergeBlock,
			def: DefPullReq{
				Merge: DefMerge{
					Block: true,
				},
			},
			in: MergeVerifyInput{
				Method: enum.MergeMethodMerge,
				PullReq: &types.PullReq{
					TargetBranch: "abc",
				},
			},
			expCodes:  []string{codePullReqMergeBlock},
			expParams: [][]any{{"abc"}},
			expOut: MergeVerifyOutput{
				AllowedMethods: enum.MergeMethods,
			},
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
