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
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/gitrpc"
	gitrpcenum "github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type MergeInput struct {
	Method    enum.MergeMethod `json:"method"`
	SourceSHA string           `json:"source_sha"`
}

// Merge merges the pull request.
//
//nolint:gocognit
func (c *Controller) Merge(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *MergeInput,
) (*types.MergeResponse, *types.MergeViolations, error) {
	method, ok := in.Method.Sanitize()
	if !ok {
		return nil, nil, usererror.BadRequest(
			fmt.Sprintf("wrong merge method type: %s", in.Method))
	}
	in.Method = method

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	// if two requests for merging comes at the same time then mutex will lock
	// first one and second one will wait, when first one is done then second one
	// continue with latest data from db with state merged and return error that
	// pr is already merged.
	mutex, err := c.newMutexForPR(targetRepo.GitUID, 0) // 0 means locks all PRs for this repo
	if err != nil {
		return nil, nil, err
	}
	err = mutex.Lock(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = mutex.Unlock(ctx)
	}()

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if pr.Merged != nil {
		return nil, nil, usererror.BadRequest("Pull request already merged")
	}

	if pr.State != enum.PullReqStateOpen {
		return nil, nil, usererror.BadRequest("Pull request must be open")
	}

	/*
		if pr.SourceSHA != in.SourceSHA {
			return nil, nil,
				usererror.BadRequest("A newer commit is available. Only the latest commit can be merged.")
		}
	*/

	if pr.IsDraft {
		return nil, nil, usererror.BadRequest(
			"Draft pull requests can't be merged. Clear the draft flag first.",
		)
	}

	reviewers, err := c.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load list of reviwers: %w", err)
	}

	targetWriteParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, targetRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	sourceRepo := targetRepo
	sourceWriteParams := targetWriteParams
	if pr.SourceRepoID != pr.TargetRepoID {
		sourceWriteParams, err = controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, sourceRepo)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
		}

		sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get source repository: %w", err)
		}
	}

	isSpaceOwner, err := apiauth.IsSpaceAdmin(ctx, c.authorizer, session, targetRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine if the user is space admin: %w", err)
	}

	checkResults, err := c.checkStore.ListResults(ctx, targetRepo.ID, pr.SourceSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list status checks: %w", err)
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, targetRepo.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	ownersForPR, err := c.codeOwners.GetApplicableCodeOwnersForPR(ctx, sourceRepo, pr)
	if codeowners.IsTooLargeError(err) {
		return nil, nil, usererror.UnprocessableEntityf(err.Error())
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find codeOwners for PR: %w", err)
	}

	codeOwnerWithApproval, err := c.codeOwners.Evaluate(ctx, ownersForPR, reviewers)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get code owners with approval: %w", err)
	}

	ruleOut, violations, err := protectionRules.CanMerge(ctx, protection.CanMergeInput{
		Actor:        &session.Principal,
		IsSpaceOwner: isSpaceOwner,
		TargetRepo:   targetRepo,
		SourceRepo:   sourceRepo,
		PullReq:      pr,
		Reviewers:    reviewers,
		Method:       in.Method,
		CheckResults: checkResults,
		CodeOwners:   codeOwnerWithApproval,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}
	if protection.IsCritical(violations) {
		return nil, &types.MergeViolations{RuleViolations: violations}, nil
	}

	// TODO: for forking merge title might be different?
	var mergeTitle string
	if in.Method == enum.MergeMethod(gitrpcenum.MergeMethodSquash) {
		mergeTitle = fmt.Sprintf("%s (#%d)", pr.Title, pr.Number)
	} else {
		mergeTitle = fmt.Sprintf("Merge branch '%s' of %s (#%d)", pr.SourceBranch, sourceRepo.Path, pr.Number)
	}

	now := time.Now()
	var mergeOutput gitrpc.MergeOutput
	mergeOutput, err = c.gitRPCClient.Merge(ctx, &gitrpc.MergeParams{
		WriteParams:     targetWriteParams,
		BaseBranch:      pr.TargetBranch,
		HeadRepoUID:     sourceRepo.GitUID,
		HeadBranch:      pr.SourceBranch,
		Title:           mergeTitle,
		Message:         "",
		Committer:       rpcIdentityFromPrincipal(bootstrap.NewSystemServiceSession().Principal),
		CommitterDate:   &now,
		Author:          rpcIdentityFromPrincipal(session.Principal),
		AuthorDate:      &now,
		RefType:         gitrpcenum.RefTypeBranch,
		RefName:         pr.TargetBranch,
		HeadExpectedSHA: in.SourceSHA,
		Method:          gitrpcenum.MergeMethod(in.Method),
	})
	if err != nil {
		if gitrpc.ErrorStatus(err) == gitrpc.StatusNotMergeable {
			return &types.MergeResponse{
				SHA:            "",
				BranchDeleted:  false,
				ConflictFiles:  gitrpc.AsConflictFilesError(err),
				RuleViolations: violations,
			}, nil, nil
			// TODO: This should be the response in case of a merge conflict.
			// TODO: Remove the ConflictFiles field from types.MergeResponse.
			/*
				return nil, &types.MergeViolations{
					ConflictFiles:  gitrpc.AsConflictFilesError(err),
					RuleViolations: violations,
				}, nil
			*/
		}
		return nil, nil, fmt.Errorf("merge check execution failed: %w", err)
	}

	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.State = enum.PullReqStateMerged

		nowMilli := now.UnixMilli()
		pr.Merged = &nowMilli
		pr.MergedBy = &session.Principal.ID
		pr.MergeMethod = &in.Method

		// update all Merge specific information (might be empty if previous merge check failed)
		pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
		pr.MergeTargetSHA = &mergeOutput.BaseSHA
		pr.MergeBaseSHA = mergeOutput.MergeBaseSHA
		pr.MergeSHA = &mergeOutput.MergeSHA
		pr.MergeConflicts = nil

		pr.ActivitySeq++ // because we need to write the activity entry
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	activityPayload := &types.PullRequestActivityPayloadMerge{
		MergeMethod: in.Method,
		MergeSHA:    mergeOutput.MergeSHA,
		TargetSHA:   mergeOutput.BaseSHA,
		SourceSHA:   mergeOutput.HeadSHA,
	}
	if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, activityPayload); errAct != nil {
		// non-critical error
		log.Ctx(ctx).Err(errAct).Msgf("failed to write pull req merge activity")
	}

	c.eventReporter.Merged(ctx, &pullreqevents.MergedPayload{
		Base:        eventBase(pr, &session.Principal),
		MergeMethod: in.Method,
		MergeSHA:    mergeOutput.MergeSHA,
		TargetSHA:   mergeOutput.BaseSHA,
		SourceSHA:   mergeOutput.HeadSHA,
	})

	var branchDeleted bool
	if ruleOut.DeleteSourceBranch {
		errDelete := c.gitRPCClient.DeleteBranch(ctx, &gitrpc.DeleteBranchParams{
			WriteParams: sourceWriteParams,
			BranchName:  pr.SourceBranch,
		})
		if errDelete != nil {
			// non-critical error
			log.Ctx(ctx).Err(errDelete).Msgf("failed to delete source branch after merging")
		} else {
			branchDeleted = true
		}
	}

	return &types.MergeResponse{
		SHA:            mergeOutput.MergeSHA,
		BranchDeleted:  branchDeleted,
		RuleViolations: violations,
	}, nil, nil
}
