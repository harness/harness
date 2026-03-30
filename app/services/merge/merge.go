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

package merge

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var (
	ErrNotEligible      = errors.New("not eligible")
	ErrRuleViolation    = errors.New("rule violation error")
	ErrConflict         = errors.New("conflict error")
	ErrMethodNotAllowed = errors.New("merge method not allowed")
)

// Timeout is the max time we give a merge to succeed.
const Timeout = 3 * time.Minute

func (s *Service) autoMerge(ctx context.Context, pr *types.PullReq) error {
	if !isEligibleForAutoMerge(pr) {
		return nil
	}

	log := log.Ctx(ctx).With().
		Int64("repo_id", pr.TargetRepoID).
		Int64("pullreq_id", pr.ID).
		Int64("pullreq_number", pr.Number).
		Logger()

	autoMerge, err := s.autoMergeStore.Find(ctx, pr.ID)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		log.Warn().Msg("pull request marked for auto merging, but no auto merge entry found in DB")
		return events.NewDiscardEventError(err)
	}
	if err != nil {
		log.Warn().Err(err).Msg("failed to find auto merge")
		return err
	}

	principal, err := s.principalStore.Find(ctx, autoMerge.RequestedBy)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find principal by ID for auto merge")
		return err
	}

	input := types.AutoMergeInput{
		Principal:    *principal,
		MergeMethod:  autoMerge.MergeMethod,
		Title:        autoMerge.Title,
		Message:      autoMerge.Message,
		DeleteBranch: autoMerge.DeleteBranch,
	}

	prMerged, branchDeleted, err := s.Merge(ctx, pr, input)
	if errors.Is(err, ErrMethodNotAllowed) {
		log.Debug().
			Msg("auto-merge pull request could not be performed because merge method is not allowed by rules")

		err = s.disableAutoMerge(ctx, pr.ID, autoMerge.MergeMethod)
		if err != nil {
			log.Warn().Err(err).Msg("failed to disable auto merge for PR")
		}
		return nil
	}
	if errors.Is(err, ErrNotEligible) {
		log.Debug().
			Msg("auto-merge pull request could not be performed because the pull request is not eligible")
		return events.NewDiscardEventError(err)
	}
	if errors.Is(err, ErrRuleViolation) {
		log.Debug().
			Msg("auto-merge pull request could not be performed because of a rule violation")
		return events.NewDiscardEventError(err)
	}
	if errors.Is(err, ErrConflict) {
		log.Debug().
			Msg("auto-merge pull request could not be performed because of a merge conflict")
		return events.NewDiscardEventError(err)
	}
	if err != nil {
		log.Warn().Err(err).Msg("failed to auto-merge pull request")
		return err
	}

	log.Info().
		Str("merge_method", string(*prMerged.MergeMethod)).
		Bool("branch_deleted", branchDeleted).
		Msg("successfully auto-merged pull request")

	return nil
}

// Merge merges the provided pull request. It returns success (not error) only if the merging succeeded.
// If the merging succeeded the relevant git references would be updated and the PR would be marked as merged in the DB.
// If the merging failed because of the pull request state, rules or a conflict the error would be one of the
// ErrNotEligible, ErrRuleViolation or ErrConflict, respectively.
func (s *Service) Merge(
	ctx context.Context,
	pr *types.PullReq,
	input types.AutoMergeInput,
) (*types.PullReq, bool, error) {
	if pr.State != enum.PullReqStateOpen || pr.IsDraft {
		return nil, false, fmt.Errorf("can merge only open, non-draft pull requests: %w", ErrNotEligible)
	}

	if pr.SourceRepoID == nil {
		return nil, false, fmt.Errorf("can't merge a PR without source repo: %w", ErrNotEligible)
	}

	targetRepo, err := s.repoFinder.FindByID(ctx, pr.TargetRepoID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find target repo: %w", err)
	}

	var sourceRepo *types.RepositoryCore

	unlock, err := s.locker.LockPR(ctx, targetRepo.ID, 0, Timeout) // 0 means locking repo level for prs
	if err != nil {
		return nil, false, fmt.Errorf("failed to lock repository for pull request merge: %w", err)
	}
	defer unlock()

	switch {
	case pr.SourceRepoID == nil:
		// the source repo is purged
	case *pr.SourceRepoID != pr.TargetRepoID:
		// if the source repo is nil, it's deleted
		sourceRepo, err = s.repoFinder.FindByID(ctx, *pr.SourceRepoID)
		if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return nil, false, fmt.Errorf("failed to get source repository: %w", err)
		}
	default:
		sourceRepo = targetRepo
	}

	targetBranch, err := s.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: git.ReadParams{RepoUID: targetRepo.GitUID},
		BranchName: pr.TargetBranch,
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to get pull request target branch: %w", err)
	}

	targetSHA := targetBranch.Branch.SHA

	reviewers, err := s.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load list of reviwers: %w", err)
	}

	protectionRules, err := s.protectionManager.ListRepoBranchRules(ctx, pr.TargetRepoID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	checkResults, err := s.checkStore.ListResults(ctx, pr.TargetRepoID, pr.SourceSHA)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list status checks: %w", err)
	}

	codeOwnerWithApproval, err := s.codeOwners.Evaluate(ctx, targetRepo, pr, reviewers)
	if err != nil && !errors.Is(err, codeowners.ErrNotFound) {
		return nil, false, fmt.Errorf("CODEOWNERS evaluation failed: %w", err)
	}

	ruleOut, violations, err := protectionRules.MergeVerify(ctx, protection.MergeVerifyInput{
		ResolveUserGroupIDs: s.userGroupService.ListUserIDsByGroupIDs,
		MapUserGroupIDs:     s.userGroupService.MapGroupIDsToPrincipals,
		Actor:               &input.Principal,
		AllowBypass:         false,
		IsRepoOwner:         false,
		TargetRepo:          targetRepo,
		SourceRepo:          sourceRepo,
		PullReq:             pr,
		Reviewers:           reviewers,
		Method:              input.MergeMethod,
		CheckResults:        checkResults,
		CodeOwners:          codeOwnerWithApproval,
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if violations != nil {
		if !slices.Contains(ruleOut.AllowedMethods, input.MergeMethod) {
			return nil, false, ErrMethodNotAllowed
		}
		return nil, false, ErrRuleViolation
	}

	// only delete the source branch if it's the source repository is the same as the target repository.
	deleteSourceBranch :=
		pr.SourceRepoID != nil && pr.TargetRepoID == *pr.SourceRepoID &&
			(ruleOut.DeleteSourceBranch || input.DeleteBranch)

	principalInfo := input.Principal.ToPrincipalInfo()

	mergeInput, err := s.PreparePullReqMergeInput(
		pr,
		sourceRepo,
		targetSHA,
		principalInfo,
		input.MergeMethod,
		input.Title,
		input.Message,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to prepare merge input: %w", err)
	}

	targetWriteParams, err := s.createRPCWriteParams(ctx, principalInfo, targetRepo.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	now := time.Now()
	mergeOutput, err := s.git.Merge(ctx, &git.MergeParams{
		WriteParams:   targetWriteParams,
		BaseSHA:       targetSHA,
		HeadSHA:       mergeInput.SourceSHA,
		Message:       mergeInput.CommitMessage,
		Committer:     mergeInput.Committer,
		CommitterDate: &now,
		Author:        mergeInput.Author,
		AuthorDate:    &now,
		Refs:          mergeInput.RefUpdates,
		Method:        gitenum.MergeMethod(input.MergeMethod),
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to merge pull request: %w", err)
	}

	if mergeOutput.MergeSHA.IsEmpty() {
		return nil, false, ErrConflict
	}

	// Update pull request in the database
	pr, seqBranchDeleted, err := s.DatabaseUpdate(
		ctx,
		pr,
		input.MergeMethod,
		mergeOutput,
		principalInfo,
		false,
		"",
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to update pull request after automerge: %w", err)
	}

	// Try to delete the source branch and insert pull request activity for it.
	var branchDeleted bool
	if deleteSourceBranch {
		branchDeleted = s.DeleteBranchTry(ctx, pr, principalInfo, seqBranchDeleted)
	}

	// Publish pull request merge events
	s.Publish(
		ctx,
		pr,
		targetRepo,
		input.MergeMethod,
		mergeOutput,
		principalInfo,
	)

	return pr, branchDeleted, nil
}

func (s *Service) disableAutoMerge(ctx context.Context, prID int64, method enum.MergeMethod) error {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

	var (
		targetRepo *types.RepositoryCore
		pr         *types.PullReq
	)

	err := controller.TxOptLock(ctx, s.tx, func(ctx context.Context) error {
		var err error

		pr, err = s.pullreqStore.Find(ctx, prID)
		if err != nil {
			return fmt.Errorf("failed to find pull request by ID: %w", err)
		}

		// Just to be safe, make sure the PR has not been already merged or auto-merge disabled.
		if pr.Merged != nil || pr.State != enum.PullReqStateOpen || pr.IsDraft ||
			pr.SubState != enum.PullReqSubStateAutoMerge {
			return events.NewDiscardEventError(errors.New("PR not open or auto-merge already disabled"))
		}

		pr.SubState = enum.PullReqSubStateNone
		pr.ActivitySeq++

		targetRepo, err = s.repoFinder.FindByID(ctx, pr.TargetRepoID)
		if err != nil {
			return fmt.Errorf("failed to get target repository: %w", err)
		}

		err = s.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		_, err = s.autoMergeStore.Delete(ctx, pr.ID)
		if err != nil {
			return fmt.Errorf("failed to update auto merge state: %w", err)
		}

		activityPayload := &types.PullRequestActivityPayloadAutoMergeDisabled{
			MergeMethod: method,
		}

		_, err = s.activityStore.CreateWithPayload(ctx, pr, systemPrincipal.ID, activityPayload, nil)
		if err != nil {
			return fmt.Errorf("failed to add disable auto-merge activity: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to disable auto merge for the pull request: %w", err)
	}

	s.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqAutoMergeDisabled, pr)

	return nil
}
