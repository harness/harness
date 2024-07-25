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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

type MergeInput struct {
	Method      enum.MergeMethod `json:"method"`
	SourceSHA   string           `json:"source_sha"`
	Title       string           `json:"title"`
	Message     string           `json:"message"`
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

	// cleanup title / message (NOTE: git doesn't support white space only)
	in.Title = strings.TrimSpace(in.Title)
	in.Message = strings.TrimSpace(in.Message)

	if in.Method == enum.MergeMethodRebase && (in.Title != "" || in.Message != "") {
		return usererror.BadRequest("rebase doesn't support customizing commit title and message")
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

	requiredPermission := enum.PermissionRepoPush
	if in.DryRun {
		requiredPermission = enum.PermissionRepoView
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, requiredPermission)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	// the max time we give a merge to succeed
	const timeout = 3 * time.Minute

	var lockID int64 // 0 means locking repo level for prs (for actual merging)
	if in.DryRun {
		lockID = pullreqNum // dryrun doesn't need repo level lock
	}

	// if two requests for merging comes at the same time then unlock will lock
	// first one and second one will wait, when first one is done then second one
	// continue with latest data from db with state merged and return error that
	// pr is already merged.
	unlock, err := c.locker.LockPR(
		ctx,
		targetRepo.ID,
		lockID,
		timeout+30*time.Second, // add 30s to the lock to give enough time for pre + post merge
	)
	if err != nil {
		return nil, nil, err
	}
	defer unlock()

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

	if pr.IsDraft && !in.DryRun {
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
	if err != nil && !errors.Is(err, codeowners.ErrNotFound) {
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

	// we want to complete the merge independent of request cancel - start with new, time restricted context.
	// TODO: This is a small change to reduce likelihood of dirty state.
	// We still require a proper solution to handle an application crash or very slow execution times
	// (which could cause an unlocking pre operation completion).
	ctx, cancel := context.WithTimeout(
		contextutil.WithNewValues(context.Background(), ctx),
		timeout,
	)
	defer cancel()

	//nolint:nestif
	if in.DryRun {
		// As the merge API is always executed under a global lock, we use the opportunity of dry-running the merge
		// to check the PR's mergeability status if it's currently "unchecked". This can happen if the target branch
		// has advanced. It's possible that the merge base commit is different too.
		// So, the next time the API gets called for the same PR the mergeability status will not be unchecked.
		// Without dry-run the execution would proceed below and would either merge the PR or set the conflict status.
		if pr.MergeCheckStatus == enum.MergeCheckStatusUnchecked {
			mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
				WriteParams:     targetWriteParams,
				BaseBranch:      pr.TargetBranch,
				HeadRepoUID:     sourceRepo.GitUID,
				HeadBranch:      pr.SourceBranch,
				RefType:         gitenum.RefTypeUndefined, // update no refs -> no commit will be created
				HeadExpectedSHA: sha.Must(in.SourceSHA),
			})
			if err != nil {
				return nil, nil, fmt.Errorf("merge check execution failed: %w", err)
			}

			pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
				if pr.SourceSHA != mergeOutput.HeadSHA.String() {
					return errors.New("source SHA has changed")
				}
				// actual merge is using a different lock - ensure we don't overwrite any merge results.
				if pr.State != enum.PullReqStateOpen {
					return usererror.BadRequest("Pull request must be open")
				}

				if len(mergeOutput.ConflictFiles) > 0 {
					pr.MergeCheckStatus = enum.MergeCheckStatusConflict
					pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
					pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
					pr.MergeSHA = nil
					pr.MergeConflicts = mergeOutput.ConflictFiles
				} else {
					pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
					pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
					pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
					pr.MergeSHA = nil // dry-run doesn't create a merge commit so output is empty.
					pr.MergeConflicts = nil
				}
				pr.Stats.DiffStats = types.NewDiffStats(
					mergeOutput.CommitCount,
					mergeOutput.ChangedFileCount,
					mergeOutput.Additions,
					mergeOutput.Deletions,
				)
				return nil
			})
			if err != nil {
				// non-critical error
				log.Ctx(ctx).Warn().Err(err).Msg("failed to update unchecked pull request")
			} else {
				if err = c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
					log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
				}
			}
		}

		// With in.DryRun=true this function never returns types.MergeViolations
		out := &types.MergeResponse{
			BranchDeleted:  ruleOut.DeleteSourceBranch,
			RuleViolations: violations,

			// values only retured by dry run
			DryRun:                              true,
			ConflictFiles:                       pr.MergeConflicts,
			AllowedMethods:                      ruleOut.AllowedMethods,
			RequiresCodeOwnersApproval:          ruleOut.RequiresCodeOwnersApproval,
			RequiresCodeOwnersApprovalLatest:    ruleOut.RequiresCodeOwnersApprovalLatest,
			RequiresCommentResolution:           ruleOut.RequiresCommentResolution,
			RequiresNoChangeRequests:            ruleOut.RequiresNoChangeRequests,
			MinimumRequiredApprovalsCount:       ruleOut.MinimumRequiredApprovalsCount,
			MinimumRequiredApprovalsCountLatest: ruleOut.MinimumRequiredApprovalsCountLatest,
		}

		return out, nil, nil
	}

	if protection.IsCritical(violations) {
		return nil, &types.MergeViolations{RuleViolations: violations}, nil
	}

	// commit details: author, committer and message

	var author *git.Identity

	switch in.Method {
	case enum.MergeMethodMerge:
		author = identityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	case enum.MergeMethodSquash:
		author = identityFromPrincipalInfo(pr.Author)
	case enum.MergeMethodRebase:
		author = nil // Not important for the rebase merge: the author info in the commits will be preserved.
	}

	var committer *git.Identity

	switch in.Method {
	case enum.MergeMethodMerge, enum.MergeMethodSquash:
		committer = identityFromPrincipalInfo(*bootstrap.NewSystemServiceSession().Principal.ToPrincipalInfo())
	case enum.MergeMethodRebase:
		committer = identityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	}

	// backfill commit title if none provided
	if in.Title == "" {
		switch in.Method {
		case enum.MergeMethodMerge:
			in.Title = fmt.Sprintf("Merge branch '%s' of %s (#%d)", pr.SourceBranch, sourceRepo.Path, pr.Number)
		case enum.MergeMethodSquash:
			in.Title = fmt.Sprintf("%s (#%d)", pr.Title, pr.Number)
		case enum.MergeMethodRebase:
			// Not used.
		}
	}

	// create merge commit(s)

	log.Ctx(ctx).Debug().Msgf("all pre-check passed, merge PR")

	now := time.Now()
	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams:     targetWriteParams,
		BaseBranch:      pr.TargetBranch,
		HeadRepoUID:     sourceRepo.GitUID,
		HeadBranch:      pr.SourceBranch,
		Title:           in.Title,
		Message:         in.Message,
		Committer:       committer,
		CommitterDate:   &now,
		Author:          author,
		AuthorDate:      &now,
		RefType:         gitenum.RefTypeBranch,
		RefName:         pr.TargetBranch,
		HeadExpectedSHA: sha.Must(in.SourceSHA),
		Method:          gitenum.MergeMethod(in.Method),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("merge check execution failed: %w", err)
	}
	//nolint:nestif
	if mergeOutput.MergeSHA.String() == "" || len(mergeOutput.ConflictFiles) > 0 {
		_, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
			if pr.SourceSHA != mergeOutput.HeadSHA.String() {
				return errors.New("source SHA has changed")
			}

			// update all Merge specific information
			pr.MergeCheckStatus = enum.MergeCheckStatusConflict
			pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
			pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
			pr.MergeSHA = nil
			pr.MergeConflicts = mergeOutput.ConflictFiles
			pr.Stats.DiffStats = types.NewDiffStats(
				mergeOutput.CommitCount,
				mergeOutput.ChangedFileCount,
				mergeOutput.Additions,
				mergeOutput.Deletions,
			)
			return nil
		})
		if err != nil {
			// non-critical error
			log.Ctx(ctx).Warn().Err(err).Msg("failed to update pull request with conflict files")
		} else {
			if err = c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
			}
		}

		return nil, &types.MergeViolations{
			ConflictFiles:  mergeOutput.ConflictFiles,
			RuleViolations: violations,
		}, nil
	}

	log.Ctx(ctx).Debug().Msgf("successfully merged PR")

	mergedBy := session.Principal.ID

	var activitySeqMerge, activitySeqBranchDeleted int64
	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.State = enum.PullReqStateMerged

		nowMilli := now.UnixMilli()
		pr.Merged = &nowMilli
		pr.MergedBy = &mergedBy
		pr.MergeMethod = &in.Method

		// update all Merge specific information (might be empty if previous merge check failed)
		// since this is the final operation on the PR, we update any sha that might've changed by now.
		pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
		pr.SourceSHA = mergeOutput.HeadSHA.String()
		pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
		pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
		pr.MergeSHA = ptr.String(mergeOutput.MergeSHA.String())
		pr.MergeConflicts = nil
		pr.Stats.DiffStats = types.NewDiffStats(
			mergeOutput.CommitCount,
			mergeOutput.ChangedFileCount,
			mergeOutput.Additions,
			mergeOutput.Deletions,
		)

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
		MergeMethod:   in.Method,
		MergeSHA:      mergeOutput.MergeSHA.String(),
		TargetSHA:     mergeOutput.BaseSHA.String(),
		SourceSHA:     mergeOutput.HeadSHA.String(),
		RulesBypassed: protection.IsBypassed(violations),
	}
	if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, mergedBy, activityPayload, nil); errAct != nil {
		// non-critical error
		log.Ctx(ctx).Err(errAct).Msgf("failed to write pull req merge activity")
	}

	c.eventReporter.Merged(ctx, &pullreqevents.MergedPayload{
		Base:        eventBase(pr, &session.Principal),
		MergeMethod: in.Method,
		MergeSHA:    mergeOutput.MergeSHA.String(),
		TargetSHA:   mergeOutput.BaseSHA.String(),
		SourceSHA:   mergeOutput.HeadSHA.String(),
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
			if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, mergedBy,
				&types.PullRequestActivityPayloadBranchDelete{SHA: in.SourceSHA}, nil); errAct != nil {
				// non-critical error
				log.Ctx(ctx).Err(errAct).
					Msgf("failed to write pull request activity for successful automatic branch delete")
			}
		}
	}

	if err = c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
	}

	return &types.MergeResponse{
		SHA:            mergeOutput.MergeSHA.String(),
		BranchDeleted:  branchDeleted,
		RuleViolations: violations,
	}, nil, nil
}
