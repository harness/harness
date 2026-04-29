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
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type MergeQueueEnableInput struct {
	Method             enum.MergeMethod `json:"method"`
	Title              string           `json:"title"`
	Message            string           `json:"message"`
	DeleteSourceBranch bool             `json:"delete_source_branch"`
	BypassRules        bool             `json:"bypass_rules"`
}

func (in *MergeQueueEnableInput) sanitize() error {
	if in.Method == "" {
		return usererror.BadRequest("Merge method must be provided.")
	}

	method, ok := in.Method.Sanitize()
	if !ok {
		return usererror.BadRequestf("Unsupported merge method: %q", in.Method)
	}

	if method == enum.MergeMethodFastForward {
		return usererror.BadRequest("Merge method cannot be fast-forward")
	}

	in.Method = method

	return nil
}

func (c *Controller) MergeQueueEnable(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *MergeQueueEnableInput,
) (*types.PullReq, *types.MergeViolations, error) {
	if err := in.sanitize(); err != nil {
		return nil, nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if pr.SubState == enum.PullReqSubStateAutoMerge {
		return nil, nil, usererror.BadRequest(
			"Enqueueing is not allowed, because the pull request has auto-merge enabled. Disable auto-merge first.")
	}

	sourceRepo := targetRepo
	if pr.SourceRepoID != nil && pr.TargetRepoID != *pr.SourceRepoID {
		sourceRepo, err = c.repoFinder.FindByID(ctx, *pr.SourceRepoID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find source repo: %w", err)
		}
	}

	protectionRules, isRepoOwner, err := c.fetchRules(ctx, session, targetRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	setup, err := protectionRules.GetMergeQueueSetup(protection.MergeQueueSetupInput{
		Repo:         targetRepo,
		TargetBranch: pr.TargetBranch,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get merge queue setup: %w", err)
	}

	if !setup.IsActive() {
		return nil, nil,
			usererror.BadRequestf("Merge queue has not been configured for branch %q.", pr.TargetBranch)
	}

	_, violations, err := c.mergeService.CheckRules(ctx, protectionRules, merge.CheckRulesInput{
		PullReq:          pr,
		TargetRepo:       targetRepo,
		SourceRepo:       sourceRepo,
		Actor:            &session.Principal,
		IsRepoOwner:      isRepoOwner,
		MergeMethod:      in.Method,
		AllowBypassRules: in.BypassRules,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if protection.IsCritical(violations) {
		return nil, &types.MergeViolations{
			RuleViolations: violations,
			Message:        protection.GenerateErrorMessageForBlockingViolations(violations),
		}, nil
	}

	pr, err = c.mergeQueueService.Enqueue(
		ctx,
		pr,
		targetRepo,
		session.Principal.ID,
		in.Method,
		in.Title,
		in.Message,
		in.DeleteSourceBranch,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enqueue pull request: %w", err)
	}

	c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqMergeQueueEnabled, pr)

	return pr, nil, nil
}
