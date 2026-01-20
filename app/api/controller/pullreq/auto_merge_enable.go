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
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type AutoMergeEnableInput struct {
	Method             enum.MergeMethod `json:"method"`
	Title              string           `json:"title"`
	Message            string           `json:"message"`
	DeleteSourceBranch bool             `json:"delete_source_branch"`
}

func (in *AutoMergeEnableInput) sanitize() error {
	if in.Method == "" {
		return usererror.BadRequest("Merge method must be provided.")
	}

	method, ok := in.Method.Sanitize()
	if !ok {
		return usererror.BadRequestf("Unsupported merge method: %q", in.Method)
	}

	in.Method = method

	// cleanup title / message (NOTE: git doesn't support white space only)
	in.Title = strings.TrimSpace(in.Title)
	in.Message = strings.TrimSpace(in.Message)

	if (in.Method == enum.MergeMethodRebase || in.Method == enum.MergeMethodFastForward) &&
		(in.Title != "" || in.Message != "") {
		return usererror.BadRequestf(
			"merge method %q doesn't support customizing commit title and message", in.Method)
	}

	return nil
}

func (c *Controller) AutoMergeEnable(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *AutoMergeEnableInput,
) (*types.AutoMergeResponse, error) {
	if err := in.sanitize(); err != nil {
		return nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	err = verifyIfAutoMergeable(pr)
	if err != nil {
		return nil, err
	}

	autoMerge := types.AutoMerge{
		PullReqID:    pr.ID,
		Requested:    time.Now().UnixMilli(),
		RequestedBy:  session.Principal.ID,
		MergeMethod:  in.Method,
		Title:        in.Title,
		Message:      in.Message,
		DeleteBranch: in.DeleteSourceBranch,
	}

	// Try to merge the pull request right now.

	prMerged, branchDeleted, err := c.mergeService.Merge(ctx, pr, types.AutoMergeInput{
		Principal:    session.Principal,
		MergeMethod:  autoMerge.MergeMethod,
		Title:        autoMerge.Title,
		Message:      autoMerge.Message,
		DeleteBranch: autoMerge.DeleteBranch,
	})
	if err != nil &&
		!errors.Is(err, merge.ErrNotEligible) &&
		!errors.Is(err, merge.ErrRuleViolation) &&
		!errors.Is(err, merge.ErrConflict) {
		return nil, fmt.Errorf("failed to merge pull request %d: %w", pr.ID, err)
	}

	// if merge succeeded we're done

	if prMerged != nil && prMerged.MergeMethod != nil && prMerged.MergeSHA != nil {
		pr = prMerged
		return &types.AutoMergeResponse{
			MergeResponse: &types.MergeResponse{
				SHA:           *pr.MergeSHA,
				BranchDeleted: branchDeleted,
			},
			Requested:    autoMerge.Requested,
			RequestedBy:  session.Principal.ToPrincipalInfo(),
			MergeMethod:  autoMerge.MergeMethod,
			Title:        autoMerge.Title,
			Message:      autoMerge.Message,
			DeleteBranch: autoMerge.DeleteBranch,
		}, nil
	}

	// otherwise add a new auto merge entry

	err = controller.TxOptLock(ctx, c.tx, func(ctx context.Context) error {
		pr, err = c.pullreqStore.Find(ctx, pr.ID)
		if err != nil {
			return fmt.Errorf("failed to find pull request by ID: %w", err)
		}

		err = verifyIfAutoMergeable(pr)
		if err != nil {
			return fmt.Errorf("pull request is not mergeable: %w", err)
		}

		pr.SubState = enum.PullReqSubStateAutoMerge
		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		err = c.autoMergeStore.Upsert(ctx, &autoMerge)
		if err != nil {
			return fmt.Errorf("failed to update auto merge state: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enable auto merge for the pull request: %w", err)
	}

	return &types.AutoMergeResponse{
		MergeResponse: nil,
		Requested:     autoMerge.Requested,
		RequestedBy:   session.Principal.ToPrincipalInfo(),
		MergeMethod:   autoMerge.MergeMethod,
		Title:         autoMerge.Title,
		Message:       autoMerge.Message,
		DeleteBranch:  autoMerge.DeleteBranch,
	}, nil
}

func verifyIfAutoMergeable(pr *types.PullReq) error {
	if pr.Merged != nil {
		return usererror.BadRequest("Pull request already merged")
	}

	if pr.State != enum.PullReqStateOpen {
		return usererror.BadRequest("Pull request must be open")
	}

	if pr.IsDraft {
		return usererror.BadRequest("Draft pull requests can't be merged. Clear the draft flag first.")
	}

	return nil
}
