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
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

type MergeInput struct {
	Method             enum.MergeMethod `json:"method"`
	SourceSHA          string           `json:"source_sha"`
	Title              string           `json:"title"`
	Message            string           `json:"message"`
	DeleteSourceBranch bool             `json:"delete_source_branch"`

	BypassRules bool `json:"bypass_rules"`
	DryRun      bool `json:"dry_run"`
	DryRunRules bool `json:"dry_run_rules"`
}

func (in *MergeInput) sanitize() error {
	if in.Method == "" && !in.DryRun && !in.DryRunRules {
		return usererror.BadRequest("Merge method must be provided if dry run is false")
	}

	if in.SourceSHA == "" {
		return usererror.BadRequest("Source SHA must be provided")
	}

	if in.Method != "" {
		method, ok := in.Method.Sanitize()
		if !ok {
			return usererror.BadRequestf("Unsupported merge method: %q", in.Method)
		}

		in.Method = method
	}

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

// backfillApprovalInfo populates principal and user group information for default reviewer approvals.
func (c *Controller) backfillApprovalInfo(
	ctx context.Context,
	approvals []*types.DefaultReviewerApprovalsResponse,
) error {
	for _, approval := range approvals {
		principalInfos, err := c.principalInfoCache.Map(ctx, approval.PrincipalIDs)
		if err != nil {
			return fmt.Errorf("failed to fetch principal infos from info cache: %w", err)
		}
		approval.PrincipalInfos = maps.Values(principalInfos)

		userGroups, err := c.userGroupStore.FindManyByIDs(ctx, approval.UserGroupIDs)
		if err != nil {
			return fmt.Errorf("failed to fetch user groups info from user group store: %w", err)
		}
		userGroupInfos := make([]*types.UserGroupInfo, 0, len(userGroups))
		for _, ug := range userGroups {
			userGroupInfos = append(userGroupInfos, ug.ToUserGroupInfo())
		}
		approval.UserGroupInfos = userGroupInfos
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
	if in.DryRunRules || in.DryRun {
		requiredPermission = enum.PermissionRepoView
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, requiredPermission)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	// the max time we give a merge to succeed
	const timeout = 3 * time.Minute

	// lock the repo only if actual git merge would be attempted
	if !in.DryRunRules {
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
			return nil, nil, fmt.Errorf("failed to lock repository for pull request merge: %w", err)
		}
		defer unlock()
	}

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

	if pr.IsDraft && !in.DryRunRules && !in.DryRun {
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

		sourceRepo, err = c.repoFinder.FindByID(ctx, pr.SourceRepoID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get source repository: %w", err)
		}
	}

	protectionRules, isRepoOwner, err := c.fetchRules(ctx, session, targetRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	checkResults, err := c.checkStore.ListResults(ctx, targetRepo.ID, in.SourceSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list status checks: %w", err)
	}

	codeOwnerWithApproval, err := c.codeOwners.Evaluate(ctx, sourceRepo, pr, reviewers)
	// check for error and ignore if it is codeowners file not found else throw error
	if err != nil && !errors.Is(err, codeowners.ErrNotFound) {
		return nil, nil, fmt.Errorf("CODEOWNERS evaluation failed: %w", err)
	}

	ruleOut, violations, err := protectionRules.MergeVerify(ctx, protection.MergeVerifyInput{
		ResolveUserGroupIDs: c.userGroupService.ListUserIDsByGroupIDs,
		MapUserGroupIDs:     c.userGroupService.MapGroupIDsToPrincipals,
		Actor:               &session.Principal,
		AllowBypass:         in.BypassRules,
		IsRepoOwner:         isRepoOwner,
		TargetRepo:          targetRepo,
		SourceRepo:          sourceRepo,
		PullReq:             pr,
		Reviewers:           reviewers,
		Method:              in.Method,
		CheckResults:        checkResults,
		CodeOwners:          codeOwnerWithApproval,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	deleteSourceBranch := in.DeleteSourceBranch || ruleOut.DeleteSourceBranch

	if in.DryRunRules {
		err := c.backfillApprovalInfo(ctx, ruleOut.DefaultReviewerApprovals)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to populate approval info for default reviewers: %w", err)
		}

		return &types.MergeResponse{
			BranchDeleted:  deleteSourceBranch,
			RuleViolations: violations,

			DryRunRules:                         true,
			AllowedMethods:                      ruleOut.AllowedMethods,
			RequiresCodeOwnersApproval:          ruleOut.RequiresCodeOwnersApproval,
			RequiresCodeOwnersApprovalLatest:    ruleOut.RequiresCodeOwnersApprovalLatest,
			RequiresCommentResolution:           ruleOut.RequiresCommentResolution,
			RequiresNoChangeRequests:            ruleOut.RequiresNoChangeRequests,
			MinimumRequiredApprovalsCount:       ruleOut.MinimumRequiredApprovalsCount,
			MinimumRequiredApprovalsCountLatest: ruleOut.MinimumRequiredApprovalsCountLatest,
			DefaultReviewerApprovals:            ruleOut.DefaultReviewerApprovals,
		}, nil, nil
	}

	// we want to complete the merge independent of request cancel - start with new, time restricted context.
	// TODO: This is a small change to reduce likelihood of dirty state.
	// We still require a proper solution to handle an application crash or very slow execution times
	// (which could cause an unlocking pre operation completion).
	ctx, cancel := contextutil.WithNewTimeout(ctx, timeout)
	defer cancel()

	//nolint:nestif
	if in.DryRun {
		// As the merge API is always executed under a global lock, we use the opportunity of dry-running the merge
		// to check the PR's mergeability status if it's currently "unchecked". This can happen if the target branch
		// has advanced. It's possible that the merge base commit is different too.
		// So, the next time the API gets called for the same PR the mergeability status will not be unchecked.
		// Without dry-run the execution would proceed below and would either merge the PR or set the conflict status.

		var mergeOutput git.MergeOutput

		// We distinguish two types when checking mergeability: Rebase and Non-Rebase.
		// * Merge methods Merge and Squash will always have the same results.
		// * Merge method Rebase is special because it must always check all commits, one at a time.
		// * Merge method Fast-Forward can never have conflicts,
		//   but for it the merge base SHA must be equal to target branch SHA.
		// The result of the tests will be stored (think cached) in the database for these two types
		// in the fields merge_check_status and rebase_check_status.

		if in.Method == "" {
			in.Method = enum.MergeMethodMerge
		}

		checkMergeability := func(method enum.MergeMethod) bool {
			switch method {
			case enum.MergeMethodMerge, enum.MergeMethodSquash:
				return pr.MergeCheckStatus == enum.MergeCheckStatusUnchecked
			case enum.MergeMethodRebase:
				return pr.RebaseCheckStatus == enum.MergeCheckStatusUnchecked
			case enum.MergeMethodFastForward:
				// Always check for ff merge. There can never be conflicts,
				// but we are interested in if it returns the conflict error and merge-output data.
				return true
			default:
				return true // should not happen
			}
		}(in.Method)

		if checkMergeability {
			// for merge-check we can skip git hooks explicitly (we don't update any refs anyway)
			writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, c.urlProvider, session, targetRepo)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
			}

			mergeOutput, err = c.git.Merge(ctx, &git.MergeParams{
				WriteParams:     writeParams,
				BaseBranch:      pr.TargetBranch,
				HeadRepoUID:     sourceRepo.GitUID,
				HeadBranch:      pr.SourceBranch,
				Refs:            nil, // update no refs -> no commit will be created
				HeadExpectedSHA: sha.Must(in.SourceSHA),
				Method:          gitenum.MergeMethod(in.Method),
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed merge check with method=%s: %w", in.Method, err)
			}

			pr, err = c.pullreqStore.UpdateMergeCheckMetadataOptLock(ctx, pr, func(pr *types.PullReq) error {
				if pr.SourceSHA != mergeOutput.HeadSHA.String() {
					return errors.New("source SHA has changed")
				}
				// actual merge is using a different lock - ensure we don't overwrite any merge results.
				if pr.State != enum.PullReqStateOpen {
					return usererror.BadRequest("Pull request must be open")
				}

				pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
				pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
				pr.MergeSHA = nil // dry-run doesn't create a merge commit so output is empty.

				pr.UpdateMergeOutcome(in.Method, mergeOutput.ConflictFiles)

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
				c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)
			}
		}

		var conflicts []string
		if in.Method == enum.MergeMethodRebase {
			conflicts = pr.RebaseConflicts
		} else {
			conflicts = pr.MergeConflicts
		}

		err := c.backfillApprovalInfo(ctx, ruleOut.DefaultReviewerApprovals)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to populate approval info for default reviewers: %w", err)
		}

		// With in.DryRun=true this function never returns types.MergeViolations
		out := &types.MergeResponse{
			BranchDeleted:  deleteSourceBranch,
			RuleViolations: violations,

			// values only returned by dry run
			DryRun:                              true,
			Mergeable:                           len(conflicts) == 0,
			ConflictFiles:                       conflicts,
			AllowedMethods:                      ruleOut.AllowedMethods,
			RequiresCodeOwnersApproval:          ruleOut.RequiresCodeOwnersApproval,
			RequiresCodeOwnersApprovalLatest:    ruleOut.RequiresCodeOwnersApprovalLatest,
			RequiresCommentResolution:           ruleOut.RequiresCommentResolution,
			RequiresNoChangeRequests:            ruleOut.RequiresNoChangeRequests,
			MinimumRequiredApprovalsCount:       ruleOut.MinimumRequiredApprovalsCount,
			MinimumRequiredApprovalsCountLatest: ruleOut.MinimumRequiredApprovalsCountLatest,
			DefaultReviewerApprovals:            ruleOut.DefaultReviewerApprovals,
		}

		return out, nil, nil
	}

	if protection.IsCritical(violations) {
		sb := strings.Builder{}
		for i, ruleViolation := range violations {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(ruleViolation.Rule.Identifier)
			sb.WriteString(":[")
			for j, v := range ruleViolation.Violations {
				if j > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString(v.Code)
			}
			sb.WriteString("]")
		}

		log.Ctx(ctx).Info().Msgf("aborting pull request merge because of rule violations: %s", sb.String())

		return nil, &types.MergeViolations{
			RuleViolations: violations,
			Message:        protection.GenerateErrorMessageForBlockingViolations(violations),
		}, nil
	}

	// commit details: author, committer and message

	var author *git.Identity

	switch in.Method {
	case enum.MergeMethodMerge:
		author = controller.IdentityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	case enum.MergeMethodSquash:
		author = controller.IdentityFromPrincipalInfo(pr.Author)
	case enum.MergeMethodRebase, enum.MergeMethodFastForward:
		author = nil // Not important for these merge methods: the author info in the commits will be preserved.
	}

	var committer *git.Identity

	switch in.Method {
	case enum.MergeMethodMerge, enum.MergeMethodSquash:
		committer = controller.SystemServicePrincipalInfo()
	case enum.MergeMethodRebase:
		committer = controller.IdentityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	case enum.MergeMethodFastForward:
		committer = nil // Not important for fast-forward merge
	}

	// backfill commit title if none provided
	if in.Title == "" {
		switch in.Method {
		case enum.MergeMethodMerge:
			in.Title = fmt.Sprintf("Merge branch '%s' of %s (#%d)", pr.SourceBranch, sourceRepo.Path, pr.Number)
		case enum.MergeMethodSquash:
			in.Title = fmt.Sprintf("%s (#%d)", pr.Title, pr.Number)
		case enum.MergeMethodRebase, enum.MergeMethodFastForward:
			// Not used.
		}
	}

	// create merge commit(s)

	log.Ctx(ctx).Debug().Msgf("all pre-check passed, merge PR")

	sourceBranchSHA, err := sha.New(in.SourceSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert source SHA: %w", err)
	}

	refTargetBranch, err := git.GetRefPath(pr.TargetBranch, gitenum.RefTypeBranch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate target branch ref name: %w", err)
	}

	prNumber := strconv.FormatInt(pr.Number, 10)

	refPullReqHead, err := git.GetRefPath(prNumber, gitenum.RefTypePullReqHead)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate pull request head ref name: %w", err)
	}

	refPullReqMerge, err := git.GetRefPath(prNumber, gitenum.RefTypePullReqMerge)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate pull requert merge ref name: %w", err)
	}

	refUpdates := make([]git.RefUpdate, 0, 4)

	// Update the target branch to the result of the merge.
	refUpdates = append(refUpdates, git.RefUpdate{
		Name: refTargetBranch,
		Old:  sha.SHA{}, // don't care about the current commit SHA of the target branch.
		New:  sha.SHA{}, // update to the result of the merge.
	})

	// Make sure the PR head ref points to the correct commit after the merge.
	refUpdates = append(refUpdates, git.RefUpdate{
		Name: refPullReqHead,
		Old:  sha.SHA{}, // don't care about the old value.
		New:  sourceBranchSHA,
	})

	// Delete the PR merge reference.
	refUpdates = append(refUpdates, git.RefUpdate{
		Name: refPullReqMerge,
		Old:  sha.SHA{},
		New:  sha.Nil,
	})

	now := time.Now()
	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams:     targetWriteParams,
		BaseBranch:      pr.TargetBranch,
		HeadRepoUID:     sourceRepo.GitUID,
		HeadBranch:      pr.SourceBranch,
		Message:         git.CommitMessage(in.Title, in.Message),
		Committer:       committer,
		CommitterDate:   &now,
		Author:          author,
		AuthorDate:      &now,
		Refs:            refUpdates,
		HeadExpectedSHA: sha.Must(in.SourceSHA),
		Method:          gitenum.MergeMethod(in.Method),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("merge execution failed: %w", err)
	}
	//nolint:nestif
	if mergeOutput.MergeSHA.String() == "" || len(mergeOutput.ConflictFiles) > 0 {
		_, err = c.pullreqStore.UpdateMergeCheckMetadataOptLock(ctx, pr, func(pr *types.PullReq) error {
			if pr.SourceSHA != mergeOutput.HeadSHA.String() {
				return errors.New("source SHA has changed")
			}

			// update all Merge specific information
			pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
			pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
			pr.MergeSHA = nil
			pr.UpdateMergeOutcome(in.Method, mergeOutput.ConflictFiles)
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
			c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)
		}

		log.Ctx(ctx).Info().Msg("aborting pull request merge because of conflicts")

		return nil, &types.MergeViolations{
			ConflictFiles:  mergeOutput.ConflictFiles,
			RuleViolations: violations,
			// In case of conflicting files we prioritize those for the error message.
			Message: fmt.Sprintf("Merge blocked by conflicting files: %v", mergeOutput.ConflictFiles),
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
		pr.SourceSHA = mergeOutput.HeadSHA.String()
		pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
		pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
		pr.MergeSHA = ptr.String(mergeOutput.MergeSHA.String())
		pr.MarkAsMerged()

		bypassed := protection.IsBypassed(violations)
		pr.MergeViolationsBypassed = &bypassed

		pr.Stats.DiffStats = types.NewDiffStats(
			mergeOutput.CommitCount,
			mergeOutput.ChangedFileCount,
			mergeOutput.Additions,
			mergeOutput.Deletions,
		)

		// update sequence for PR activities
		pr.ActivitySeq++
		activitySeqMerge = pr.ActivitySeq

		if deleteSourceBranch {
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
	if deleteSourceBranch {
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

	c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)

	if protection.IsBypassed(violations) {
		err = c.auditService.Log(ctx,
			session.Principal,
			audit.NewResource(
				audit.ResourceTypeRepository,
				sourceRepo.Identifier,
				audit.RepoPath,
				sourceRepo.Path,
				audit.BypassedResourceType,
				audit.BypassedResourceTypePullRequest,
				audit.BypassedResourceName,
				strconv.FormatInt(pr.Number, 10),
				audit.ResourceName,
				fmt.Sprintf(
					audit.BypassPullReqLabelFormat,
					sourceRepo.Identifier,
					strconv.FormatInt(pr.Number, 10),
				),
				audit.BypassAction,
				audit.BypassActionMerged,
			),
			audit.ActionBypassed,
			paths.Parent(sourceRepo.Path),
			audit.WithNewObject(audit.PullRequestObject{
				PullReq:        *pr,
				RepoPath:       sourceRepo.Path,
				RuleViolations: violations,
			}),
		)
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to insert audit log for merge pull request operation: %s", err)
		}
	}

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeMergePullRequest,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      sourceRepo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   sourceRepo.ID,
			instrument.PropertyRepositoryName: sourceRepo.Identifier,
			instrument.PropertyPullRequestID:  pr.Number,
			instrument.PropertyMergeStrategy:  in.Method,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for merge pr operation: %s", err)
	}
	return &types.MergeResponse{
		SHA:            mergeOutput.MergeSHA.String(),
		BranchDeleted:  branchDeleted,
		RuleViolations: violations,
	}, nil, nil
}
