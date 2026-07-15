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

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	gitapi "github.com/harness/gitness/git/api"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

// updatePullReqTargetOnBranchDelete handles branch delete events.
// For every open pull request targeting the deleted branch, it updates the target
// to point to the repository's default branch.
func (s *Service) updatePullReqTargetOnBranchDelete(
	ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload],
) error {
	branch, err := getBranchFromRef(event.Payload.Ref)
	if err != nil {
		return events.NewDiscardEventError(
			fmt.Errorf("failed to parse branch name from ref: %w", err),
		)
	}

	const largeLimit = 1000000
	pullreqList, err := s.pullreqStore.List(ctx, &types.PullReqFilter{
		Page:         0,
		Size:         largeLimit,
		TargetRepoID: event.Payload.RepoID,
		TargetBranch: branch,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
		Sort:         enum.PullReqSortNumber,
		Order:        enum.OrderAsc,
	})
	if err != nil {
		return fmt.Errorf("failed to get list of open pull requests targeting deleted branch %s: %w", branch, err)
	}

	// No PRs to update
	if len(pullreqList) == 0 {
		return nil
	}

	// Get the repository to find its default branch
	repo, err := s.repoStore.Find(ctx, event.Payload.RepoID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Validate we can retarget PRs to the default branch
	if repo.DefaultBranch == "" {
		log.Ctx(ctx).Warn().
			Str("branch", branch).
			Msg("repository has no default branch, cannot retarget PRs")
		return nil
	}
	if repo.DefaultBranch == branch {
		log.Ctx(ctx).Warn().
			Str("branch", branch).
			Msg("default branch was deleted, cannot update PRs")
		return nil
	}

	log.Ctx(ctx).Info().
		Int64("repo_id", event.Payload.RepoID).
		Str("deleted_branch", branch).
		Str("default_branch", repo.DefaultBranch).
		Int("pr_count", len(pullreqList)).
		Msg("retargeting PRs from deleted branch to default branch")

	// Update each PR to target the default branch
	for _, pr := range pullreqList {
		if err := s.updatePRToDefaultBranch(ctx, pr, repo, branch, event.Payload.PrincipalID); err != nil {
			log.Ctx(ctx).Err(err).
				Int64("pullreq", pr.Number).
				Str("deleted_branch", branch).
				Str("default_branch", repo.DefaultBranch).
				Msg("failed to update PR target after branch deletion")
		}
	}

	return nil
}

// updatePRToDefaultBranch updates a single PR
// to target the repository's default branch.
func (s *Service) updatePRToDefaultBranch(
	ctx context.Context,
	pr *types.PullReq,
	repo *types.Repository,
	deletedBranch string,
	principalID int64,
) error {
	readParams := git.CreateReadParams(repo)

	// Get the default branch's latest commit
	defaultBranchRef, err := s.git.GetRef(ctx, git.GetRefParams{
		ReadParams: readParams,
		Name:       repo.DefaultBranch,
		Type:       gitenum.RefTypeBranch,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve default branch reference: %w", err)
	}

	defaultBranchSHA := defaultBranchRef.SHA // the latest commit on the default branch

	// Calculate the new merge base. Example:
	// main:    A -> B -> C -> D
	// PR:      A -> B -> E -> F
	// Merge base = B (last common commit)
	// PR has new commits: E, F
	mergeBaseInfo, err := s.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: readParams,
		Ref1:       pr.SourceSHA,
		Ref2:       defaultBranchSHA.String(),
	})
	if errors.IsInvalidArgument(err) || gitapi.IsUnrelatedHistoriesError(err) || gitapi.IsMergeBaseNonUniqueError(err) {
		// Non-unique merge base, unrelated histories, or invalid argument - close the PR
		log.Ctx(ctx).Warn().
			Int64("pullreq", pr.Number).
			Str("source_sha", pr.SourceSHA).
			Str("target_sha", defaultBranchSHA.String()).
			Err(err).
			Msg("cannot calculate merge base with default branch, closing PR")

		sourceSHA, err := sha.New(pr.SourceSHA)
		if err != nil {
			return fmt.Errorf("failed to parse PR source SHA %q: %w", pr.SourceSHA, err)
		}

		// Write a TargetBranchDeleted activity before closing so it appears first on the
		// timeline: target-branch-deleted → non-unique-merge-base → PR closed.
		s.writeTargetBranchDeletedActivity(ctx, pr, principalID, deletedBranch)

		err = s.CloseBecauseNonUniqueMergeBase(ctx, defaultBranchSHA, sourceSHA, pr)
		if errors.Is(err, ErrPullReqNotOpen) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to close pull request after merge base error: %w", err)
		}

		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get merge base with default branch: %w", err)
	}

	// If the merge base is the same as the source SHA, this PR has no new commits
	if mergeBaseInfo.MergeBaseSHA.String() == pr.SourceSHA {
		log.Ctx(ctx).Warn().
			Int64("pullreq", pr.Number).
			Str("deleted_branch", deletedBranch).
			Msg("PR has no new commits after updating to default branch, closing instead")

		// Close the PR since it has no changes to merge
		return s.closePRWithNoChanges(ctx, pr, repo, deletedBranch, principalID, mergeBaseInfo.MergeBaseSHA.String())
	}

	// Calculate new diff stats
	diffStats, err := s.git.DiffStats(ctx, &git.DiffParams{
		ReadParams: readParams,
		BaseRef:    mergeBaseInfo.MergeBaseSHA.String(),
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return fmt.Errorf("failed to get diff stats: %w", err)
	}

	oldTargetBranch := pr.TargetBranch
	oldMergeBaseSHA := pr.MergeBaseSHA

	// Update the PR in the database
	pr, err = s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		// Avoid racing conditions with merge
		if pr.State != enum.PullReqStateOpen {
			return ErrPullReqNotOpen
		}

		pr.ActivitySeq++

		// Update to target the default branch
		pr.TargetBranch = repo.DefaultBranch
		pr.MergeBaseSHA = mergeBaseInfo.MergeBaseSHA.String()
		pr.MergeTargetSHA = ptr.String(defaultBranchSHA.String())

		// Reset merge-check fields for new run
		pr.MergeSHA = nil // nil means: "needs to be recalculated"
		pr.Stats.DiffStats = types.NewDiffStats(
			diffStats.Commits,
			diffStats.FilesChanged,
			diffStats.Additions,
			diffStats.Deletions,
		)
		pr.MarkAsMergeUnchecked()

		return nil
	})
	if errors.Is(err, ErrPullReqNotOpen) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to update PR after branch deletion: %w", err)
	}

	// Use a non-cancellable context to ensure subsequent ops complete
	// even if the caller's context is cancelled.
	ctxNoCancel := context.WithoutCancel(ctx)

	// Create activity entry for the target branch change
	payload := &types.PullRequestActivityPayloadTargetBranchDeleted{
		OldTargetBranch: oldTargetBranch,
		NewTargetBranch: repo.DefaultBranch,
		OldMergeBaseSHA: oldMergeBaseSHA,
		NewMergeBaseSHA: mergeBaseInfo.MergeBaseSHA.String(),
	}

	_, err = s.activityStore.CreateWithPayload(ctxNoCancel, pr, principalID, payload, nil)
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).
			Int64("pullreq", pr.Number).
			Msg("failed to write pull request activity after target branch deletion")
	}

	// Update git references for the PR (delete merge ref, update head ref)
	s.deletePRMergeRef(ctxNoCancel, pr, repo)

	// Trigger merge check re-run for the new target branch
	s.pullreqEvReporter.TargetBranchChanged(ctxNoCancel, &pullreqevents.TargetBranchChangedPayload{
		Base: pullreqevents.Base{
			PullReqID:    pr.ID,
			SourceRepoID: pr.SourceRepoID,
			TargetRepoID: pr.TargetRepoID,
			PrincipalID:  principalID,
			Number:       pr.Number,
		},
		SourceSHA:       pr.SourceSHA,
		OldTargetBranch: oldTargetBranch,
		NewTargetBranch: repo.DefaultBranch,
		OldMergeBaseSHA: oldMergeBaseSHA,
		NewMergeBaseSHA: mergeBaseInfo.MergeBaseSHA.String(),
	})

	s.sseStreamer.Publish(ctxNoCancel, repo.ParentID, enum.SSETypePullReqUpdated, pr)

	log.Ctx(ctx).Info().
		Int64("pullreq", pr.Number).
		Str("old_target", oldTargetBranch).
		Str("new_target", repo.DefaultBranch).
		Str("old_merge_base", oldMergeBaseSHA).
		Str("new_merge_base", mergeBaseInfo.MergeBaseSHA.String()).
		Msg("updated PR target to default branch after branch deletion")

	return nil
}

// closePRWithNoChanges closes a PR that has no changes
// after rebasing to default branch.
func (s *Service) closePRWithNoChanges(
	ctx context.Context,
	pr *types.PullReq,
	repo *types.Repository,
	deletedBranch string,
	principalID int64,
	newMergeBaseSHA string,
) error {
	// Capture current state before updating
	oldTargetBranch := pr.TargetBranch
	oldMergeBaseSHA := pr.MergeBaseSHA
	newTargetBranch := repo.DefaultBranch

	var activitySeqTargetDeleted, activitySeqClosed int64
	pr, err := s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		if pr.State != enum.PullReqStateOpen {
			return ErrPullReqNotOpen
		}

		// Reserve two activity sequence numbers: one for target branch deletion, one for state change
		pr.ActivitySeq += 2
		activitySeqTargetDeleted = pr.ActivitySeq - 1
		activitySeqClosed = pr.ActivitySeq

		pr.TargetBranch = repo.DefaultBranch
		pr.State = enum.PullReqStateClosed
		pr.SubState = enum.PullReqSubStateNone
		pr.MergeSHA = nil
		pr.MarkAsMergeUnchecked()

		return nil
	})
	if errors.Is(err, ErrPullReqNotOpen) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to close PR with no changes: %w", err)
	}

	ctxNoCancel := context.WithoutCancel(ctx)

	_, err = s.repoStore.UpdateOptLock(ctxNoCancel, repo, func(r *types.Repository) error {
		r.NumClosedPulls++
		r.NumOpenPulls--
		return nil
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to update pull request numbers after PR close")
	}

	// Delete the PR merge git reference
	writeParams, err := createRPCSystemReferencesWriteParams(ctxNoCancel, s.urlProvider, repo.ID, repo.GitUID)
	// nolint:nestif
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to create write params for merge ref deletion")
	} else {
		mergeRefName, err := git.GetRefPath(strconv.Itoa(int(pr.Number)), gitenum.RefTypePullReqMerge)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to generate pull request merge ref name")
		} else {
			err = s.git.UpdateRef(ctxNoCancel, git.UpdateRefParams{
				WriteParams: writeParams,
				Name:        mergeRefName,
				OldValue:    sha.None, // none = don't care about value
				NewValue:    sha.Nil,  // nil = delete
			})
			if err != nil {
				log.Ctx(ctx).Err(err).
					Str("ref", mergeRefName).
					Msg("failed to delete pull request merge ref")
			}
		}
	}

	// First, record that the target branch was deleted and attempted retarget
	pr.ActivitySeq = activitySeqTargetDeleted
	targetDeletedPayload := &types.PullRequestActivityPayloadTargetBranchDeleted{
		OldTargetBranch: oldTargetBranch,
		NewTargetBranch: newTargetBranch,
		OldMergeBaseSHA: oldMergeBaseSHA,
		NewMergeBaseSHA: newMergeBaseSHA,
	}
	if _, err := s.activityStore.CreateWithPayload(ctxNoCancel, pr, principalID, targetDeletedPayload, nil); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to write target branch deleted activity")
	}

	// Then, record the state change to closed
	pr.ActivitySeq = activitySeqClosed
	stateChangePayload := &types.PullRequestActivityPayloadStateChange{
		Old:      enum.PullReqStateOpen,
		New:      enum.PullReqStateClosed,
		OldDraft: pr.IsDraft,
		NewDraft: pr.IsDraft,
	}
	if _, err := s.activityStore.CreateWithPayload(ctxNoCancel, pr, principalID, stateChangePayload, nil); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to write pull request activity for closure")
	}

	s.pullreqEvReporter.Closed(ctxNoCancel, &pullreqevents.ClosedPayload{
		Base: pullreqevents.Base{
			PullReqID:    pr.ID,
			SourceRepoID: pr.SourceRepoID,
			TargetRepoID: pr.TargetRepoID,
			PrincipalID:  principalID,
			Number:       pr.Number,
		},
		SourceSHA:    pr.SourceSHA,
		SourceBranch: pr.SourceBranch,
	})

	s.sseStreamer.Publish(ctxNoCancel, repo.ParentID, enum.SSETypePullReqUpdated, pr)

	log.Ctx(ctx).Info().
		Int64("pullreq", pr.Number).
		Str("deleted_branch", deletedBranch).
		Msg("closed PR with no changes after target branch deletion")

	return nil
}

// deletePRMergeRef deletes the stale test merge ref after the PR target branch changes.
func (s *Service) deletePRMergeRef(ctx context.Context, pr *types.PullReq, repo *types.Repository) {
	writeParams, err := createRPCSystemReferencesWriteParams(ctx, s.urlProvider, repo.ID, repo.GitUID)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to create write params for git ref updates")
		return
	}

	// refs/pullreq/{N}/merge — stale test merge commit, must be deleted
	mergeRefName, err := git.GetRefPath(strconv.Itoa(int(pr.Number)), gitenum.RefTypePullReqMerge)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to generate pull request merge ref name")
		return
	}

	err = s.git.UpdateRef(ctx, git.UpdateRefParams{
		WriteParams: writeParams,
		Name:        mergeRefName,
		OldValue:    sha.None, // none = don't care about value
		NewValue:    sha.Nil,  // nil = delete
	})
	if err != nil {
		log.Ctx(ctx).Err(err).
			Str("merge_ref", mergeRefName).
			Msg("failed to delete pull request merge ref after target change")
	}
}

// Write a TargetBranchDeleted activity so the PR timeline explains
// why the PR was closed.
func (s *Service) writeTargetBranchDeletedActivity(
	ctx context.Context,
	pr *types.PullReq,
	principalID int64,
	deletedBranch string,
) {
	ctxNoCancel := context.WithoutCancel(ctx)

	updatedPR, err := s.pullreqStore.UpdateActivitySeq(ctxNoCancel, pr)
	if err != nil {
		log.Ctx(ctx).Err(err).
			Int64("pullreq", pr.Number).
			Msg("failed to reserve activity sequence for target branch deleted activity")
		return
	}
	pr = updatedPR

	payload := &types.PullRequestActivityPayloadTargetBranchDeleted{
		OldTargetBranch: deletedBranch,
		NewTargetBranch: "",
		OldMergeBaseSHA: pr.MergeBaseSHA,
		NewMergeBaseSHA: "",
	}
	if _, err = s.activityStore.CreateWithPayload(ctxNoCancel, pr, principalID, payload, nil); err != nil {
		log.Ctx(ctx).Err(err).
			Int64("pullreq", pr.Number).
			Msg("failed to write target branch deleted activity after merge base error")
	}
}
