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
	"errors"
	"fmt"

	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/slices"
)

var TypeBranch types.RuleType = "branch"

// Branch implements protection rules for the rule type TypeBranch.
type Branch struct {
	Bypass    DefBypass    `json:"bypass"`
	PullReq   DefPullReq   `json:"pullreq"`
	Lifecycle DefLifecycle `json:"lifecycle"`
}

type Review struct {
	ReviewSHA string
	Decision  enum.PullReqReviewDecision
}

var _ Definition = (*Branch)(nil) // ensures that the Branch type implements Definition interface.

//nolint:gocognit // well aware of this
func (v *Branch) CanMerge(_ context.Context, in CanMergeInput) (CanMergeOutput, []types.RuleViolations, error) {
	var out CanMergeOutput
	var violations types.RuleViolations

	out.DeleteSourceBranch = v.PullReq.Merge.DeleteBranch

	// bypass

	if v.isBypassed(in.Actor, in.IsSpaceOwner) {
		return out, nil, nil
	}

	// pullreq.approvals

	approvedBy := make([]types.PrincipalInfo, 0, len(in.Reviewers))
	for _, reviewer := range in.Reviewers {
		if reviewer.ReviewDecision != enum.PullReqReviewDecisionApproved {
			continue
		}
		if v.PullReq.Approvals.RequireLatestCommit && reviewer.SHA != in.PullReq.SourceSHA {
			continue
		}

		approvedBy = append(approvedBy, reviewer.Reviewer)
	}

	if len(approvedBy) < v.PullReq.Approvals.RequireMinimumCount {
		violations.Addf("pullreq.approvals.require_minimum_count",
			"Insufficient number of approvals. Have %d but need at least %d.",
			len(approvedBy), v.PullReq.Approvals.RequireMinimumCount)
	}

	//nolint:nestif
	if v.PullReq.Approvals.RequireCodeOwners {
		for _, entry := range in.CodeOwners.EvaluationEntries {
			reviewDecision, approvers := getCodeOwnerApprovalStatus(entry.OwnerEvaluations)

			if reviewDecision == enum.PullReqReviewDecisionPending {
				violations.Addf("pullreq.approvals.require_code_owners",
					"Code owners approval pending for %s", entry.Pattern)
				continue
			}

			if reviewDecision == enum.PullReqReviewDecisionChangeReq {
				violations.Addf("pullreq.approvals.require_code_owners",
					"Code owners requested changes for %s", entry.Pattern)
				continue
			}
			// pull req approved. check other settings
			if !v.PullReq.Approvals.RequireLatestCommit {
				continue
			}
			// check for latest commit approved or not
			latestSHAApproved := false
			for _, approver := range approvers {
				if approver.ReviewSHA == in.PullReq.SourceSHA {
					latestSHAApproved = true
					break
				}
			}
			if !latestSHAApproved {
				violations.Addf("pullreq.approvals.require_code_owners",
					"Code owners approval pending on latest commit for %s", entry.Pattern)
			}
		}
	}

	// pullreq.comments

	if v.PullReq.Comments.RequireResolveAll && in.PullReq.UnresolvedCount > 0 {
		violations.Addf("pullreq.comments.require_resolve_all",
			"All comments must be resolved. There are %d unresolved comments.",
			in.PullReq.UnresolvedCount)
	}

	// pullreq.status_checks

	for _, requiredUID := range v.PullReq.StatusChecks.RequireUIDs {
		var succeeded bool

		for i := range in.CheckResults {
			if in.CheckResults[i].UID == requiredUID {
				succeeded = in.CheckResults[i].Status == enum.CheckStatusSuccess
				break
			}
		}

		if !succeeded {
			violations.Add("pullreq.status_checks.required_uids",
				"At least one required status check hasn't completed successfully.")
		}
	}

	// pullreq.merge

	if len(v.PullReq.Merge.StrategiesAllowed) > 0 { // Note: Empty allowed strategies list means all are allowed
		if !slices.Contains(v.PullReq.Merge.StrategiesAllowed, in.Method) {
			violations.Addf("pullreq.merge.strategies_allowed",
				"The requested merge strategy %q is not allowed. Allowed strategies are %v.",
				in.Method, v.PullReq.Merge.StrategiesAllowed)
		}
	}

	return out, []types.RuleViolations{violations}, nil
}

func (v *Branch) CanModifyRef(_ context.Context, in CanModifyRefInput) ([]types.RuleViolations, error) {
	var violations types.RuleViolations

	if v.isBypassed(in.Actor, in.IsSpaceOwner) || in.RefType != RefTypeBranch || len(in.RefNames) == 0 {
		return nil, nil
	}

	switch in.RefAction {
	case RefActionCreate:
		if v.Lifecycle.CreateForbidden {
			violations.Addf("lifecycle.create",
				"Creation of branch %q is not allowed.", in.RefNames[0])
		}
	case RefActionDelete:
		if v.Lifecycle.DeleteForbidden {
			violations.Addf("lifecycle.delete",
				"Delete of branch %q is not allowed.", in.RefNames[0])
		}
	case RefActionUpdate:
		if v.Lifecycle.UpdateForbidden {
			violations.Addf("lifecycle.update",
				"Push to branch %q is not allowed. Please use pull requests.", in.RefNames[0])
		}
	}

	return []types.RuleViolations{violations}, nil
}

func (v *Branch) isBypassed(actor *types.Principal, isSpaceOwner bool) bool {
	return actor.Admin ||
		v.Bypass.SpaceOwners && isSpaceOwner ||
		slices.Contains(v.Bypass.UserIDs, actor.ID)
}

func (v *Branch) Sanitize() error {
	if err := v.Bypass.Validate(); err != nil {
		return fmt.Errorf("bypass: %w", err)
	}

	if err := v.PullReq.Validate(); err != nil {
		return fmt.Errorf("pull request: %w", err)
	}

	if err := v.Lifecycle.Validate(); err != nil {
		return fmt.Errorf("lifecycle: %w", err)
	}

	return nil
}

type DefBypass struct {
	UserIDs     []int64 `json:"user_ids,omitempty"`
	SpaceOwners bool    `json:"space_owners,omitempty"`
}

func (v DefBypass) Validate() error {
	if err := validateIDSlice(v.UserIDs); err != nil {
		return fmt.Errorf("user IDs error: %w", err)
	}

	return nil
}

type DefApprovals struct {
	RequireCodeOwners   bool `json:"require_code_owners,omitempty"`
	RequireMinimumCount int  `json:"require_minimum_count,omitempty"`
	RequireLatestCommit bool `json:"require_latest_commit,omitempty"`
}

func (v DefApprovals) Validate() error {
	if v.RequireMinimumCount < 0 {
		return errors.New("minimum count must be zero or a positive integer")
	}

	return nil
}

type DefComments struct {
	RequireResolveAll bool `json:"require_resolve_all,omitempty"`
}

func (v DefComments) Validate() error {
	return nil
}

type DefStatusChecks struct {
	RequireUIDs []string `json:"require_uids,omitempty"`
}

func (v DefStatusChecks) Validate() error {
	if err := validateUIDSlice(v.RequireUIDs); err != nil {
		return fmt.Errorf("required UIDs error: %w", err)
	}

	return nil
}

type DefMerge struct {
	StrategiesAllowed []enum.MergeMethod `json:"strategies_allowed,omitempty"`
	DeleteBranch      bool               `json:"delete_branch,omitempty"`
}

func (v DefMerge) Validate() error {
	m := make(map[enum.MergeMethod]struct{}, 0)
	for _, strategy := range v.StrategiesAllowed {
		if _, ok := strategy.Sanitize(); !ok {
			return fmt.Errorf("unrecognized merge strategy: %s", strategy)
		}

		if _, ok := m[strategy]; ok {
			return fmt.Errorf("duplicate entry in merge strategy list: %s", strategy)
		}

		m[strategy] = struct{}{}
	}

	return nil
}

type DefPush struct {
	Block bool `json:"block,omitempty"`
}

func (v DefPush) Validate() error {
	return nil
}

type DefLifecycle struct {
	CreateForbidden bool `json:"create_forbidden,omitempty"`
	DeleteForbidden bool `json:"delete_forbidden,omitempty"`
	UpdateForbidden bool `json:"update_forbidden,omitempty"`
}

func (v DefLifecycle) Validate() error {
	return nil
}

type DefPullReq struct {
	Approvals    DefApprovals    `json:"approvals"`
	Comments     DefComments     `json:"comments"`
	StatusChecks DefStatusChecks `json:"status_checks"`
	Merge        DefMerge        `json:"merge"`
}

func (v DefPullReq) Validate() error {
	if err := v.Approvals.Validate(); err != nil {
		return fmt.Errorf("approvals: %w", err)
	}

	if err := v.Comments.Validate(); err != nil {
		return fmt.Errorf("comments: %w", err)
	}

	if err := v.StatusChecks.Validate(); err != nil {
		return fmt.Errorf("status checks: %w", err)
	}

	if err := v.Merge.Validate(); err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	return nil
}

func getCodeOwnerApprovalStatus(
	ownerStatus []codeowners.OwnerEvaluation,
) (enum.PullReqReviewDecision, []codeowners.OwnerEvaluation) {
	approvers := make([]codeowners.OwnerEvaluation, 0)
	for _, o := range ownerStatus {
		if o.ReviewDecision == enum.PullReqReviewDecisionChangeReq {
			return enum.PullReqReviewDecisionChangeReq, nil
		}
		if o.ReviewDecision == enum.PullReqReviewDecisionApproved {
			approvers = append(approvers, o)
		}
	}
	if len(approvers) > 0 {
		return enum.PullReqReviewDecisionApproved, approvers
	}
	return enum.PullReqReviewDecisionPending, nil
}
