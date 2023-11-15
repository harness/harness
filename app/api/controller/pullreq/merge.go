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
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	gittypes "github.com/harness/gitness/git/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type MergeInput struct {
	Method      enum.MergeMethod `json:"method"`
	SourceSHA   string           `json:"source_sha"`
	BypassRules bool             `json:"bypass_rules"`
	DryRun      bool             `json:"dry_run"`
}

func (in *MergeInput) sanitize() error {
	if in.Method == "" && !in.DryRun {
		return usererror.BadRequest("merge method must be provided if dry run is false")
	}

	if in.SourceSHA == "" {
		return usererror.BadRequest("source SHA must be provided")
	}

	if in.Method != "" {
		method, ok := in.Method.Sanitize()
		if !ok {
			return usererror.BadRequestf("unsupported merge method: %s", in.Method)
		}

		in.Method = method
	}

	return nil
}

// Merge merges a pull request.
//
// It supports dry running by providing the DryRun=true. Dry running can be used to find any rule violations that
// might block the merging. Dry running typically should be used with BypassRules=true.
//
// MergeMethod doesn't need to be provided for dry running. If no MergeMethod has been provided the function will
// return allowed merge methods. Rules can limit allowed merge methods.
//
// If the pull request has been successfully merged the function will return the SHA of the merge commit.
//
//nolint:gocognit,gocyclo,cyclop
func (c *Controller) Merge(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *MergeInput,
) (*types.MergeResponse, *types.MergeViolations, error) {
	if err := in.sanitize(); err != nil {
		return nil, nil, err
	}

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

	if pr.SourceSHA != in.SourceSHA {
		return nil, nil,
			usererror.BadRequest("A newer commit is available. Only the latest commit can be merged.")
	}

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

	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, targetRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	checkResults, err := c.checkStore.ListResults(ctx, targetRepo.ID, pr.SourceSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list status checks: %w", err)
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, targetRepo.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	codeOwnerWithApproval, err := c.codeOwners.Evaluate(ctx, sourceRepo, pr, reviewers)
	// check for error and ignore if it is codeowners file not found else throw error
	if err != nil && errors.AsStatus(err) != errors.StatusNotFound {
		return nil, nil, fmt.Errorf("CODEOWNERS evaluation failed: %w", err)
	}

	ruleOut, violations, err := protectionRules.MergeVerify(ctx, protection.MergeVerifyInput{
		Actor:        &session.Principal,
		AllowBypass:  in.BypassRules,
		IsRepoOwner:  isRepoOwner,
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

	if in.DryRun {
		// With in.DryRun=true this function never returns types.MergeViolations
		out := &types.MergeResponse{
			DryRun:         true,
			BranchDeleted:  ruleOut.DeleteSourceBranch,
			AllowedMethods: ruleOut.AllowedMethods,
			ConflictFiles:  pr.MergeConflicts,
			RuleViolations: violations,
		}

		// TODO: This is a temporary solution. The changes needed for the proper implementation:
		// 1) Git: Change the merge method to return SHAs (source/target/merge base) even in case of conflicts.
		// 2) Event handler: Update target and merge base SHA in the event handler even in case of merge conflicts.
		// 3) Here: Update the pull request target and merge base SHA in the DB if merge check status is unchecked.
		// 4) Remove the recheck API.
		if pr.MergeCheckStatus == enum.MergeCheckStatusUnchecked {
			_, err = c.git.Merge(ctx, &git.MergeParams{
				WriteParams:     targetWriteParams,
				BaseBranch:      pr.TargetBranch,
				HeadRepoUID:     sourceRepo.GitUID,
				HeadBranch:      pr.SourceBranch,
				HeadExpectedSHA: in.SourceSHA,
			})
			if cferr := gittypes.AsMergeConflictsError(err); cferr != nil {
				out.ConflictFiles = cferr.Files
			} else if err != nil {
				return nil, nil, fmt.Errorf("merge check execution failed: %w", err)
			}
		}

		return out, nil, nil
	}

	if protection.IsCritical(violations) {
		return nil, &types.MergeViolations{RuleViolations: violations}, nil
	}

	// TODO: for forking merge title might be different?
	var mergeTitle string
	if in.Method == enum.MergeMethod(gitenum.MergeMethodSquash) {
		mergeTitle = fmt.Sprintf("%s (#%d)", pr.Title, pr.Number)
	} else {
		mergeTitle = fmt.Sprintf("Merge branch '%s' of %s (#%d)", pr.SourceBranch, sourceRepo.Path, pr.Number)
	}

	now := time.Now()
	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams:     targetWriteParams,
		BaseBranch:      pr.TargetBranch,
		HeadRepoUID:     sourceRepo.GitUID,
		HeadBranch:      pr.SourceBranch,
		Title:           mergeTitle,
		Message:         "",
		Committer:       identityFromPrincipal(bootstrap.NewSystemServiceSession().Principal),
		CommitterDate:   &now,
		Author:          identityFromPrincipal(session.Principal),
		AuthorDate:      &now,
		RefType:         gitenum.RefTypeBranch,
		RefName:         pr.TargetBranch,
		HeadExpectedSHA: in.SourceSHA,
		Method:          gitenum.MergeMethod(in.Method),
	})
	if err != nil {
		if cf := gittypes.AsMergeConflictsError(err); cf != nil {
			//nolint: nilerr
			return nil, &types.MergeViolations{
				ConflictFiles:  cf.Files,
				RuleViolations: violations,
			}, nil
		}
		return nil, nil, fmt.Errorf("merge check execution failed: %w", err)
	}

	var activitySeqMerge, activitySeqBranchDeleted int64
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

		// update sequence for PR activities
		pr.ActivitySeq++
		activitySeqMerge = pr.ActivitySeq

		if ruleOut.DeleteSourceBranch {
			pr.ActivitySeq++
			activitySeqBranchDeleted = pr.ActivitySeq
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	pr.ActivitySeq = activitySeqMerge
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
		errDelete := c.git.DeleteBranch(ctx, &git.DeleteBranchParams{
			WriteParams: sourceWriteParams,
			BranchName:  pr.SourceBranch,
		})
		if errDelete != nil {
			// non-critical error
			log.Ctx(ctx).Err(errDelete).Msgf("failed to delete source branch after merging")
		} else {
			branchDeleted = true

			// NOTE: there is a chance someone pushed on the branch between merge and delete.
			// Either way, we'll use the SHA that was merged with for the activity to be consistent from PR perspective.
			pr.ActivitySeq = activitySeqBranchDeleted
			if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID,
				&types.PullRequestActivityPayloadBranchDelete{SHA: in.SourceSHA}); errAct != nil {
				// non-critical error
				log.Ctx(ctx).Err(errAct).Msgf("failed to write pull request activity for successful automatic branch delete")
			}
		}
	}

	if err = c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullrequesUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Msg("failed to publish PR changed event")
	}

	return &types.MergeResponse{
		SHA:            mergeOutput.MergeSHA,
		BranchDeleted:  branchDeleted,
		RuleViolations: violations,
	}, nil, nil
}
