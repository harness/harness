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
	"time"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

const (
	cancelMergeCheckKey = "cancel_merge_check_for_sha"
)

// mergeCheckOnCreated handles pull request Created events.
// It creates the PR head git ref.
func (s *Service) mergeCheckOnCreated(ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	return s.updateMergeData(
		ctx,
		event.Payload.TargetRepoID,
		event.Payload.Number,
		sha.Nil.String(),
		event.Payload.SourceSHA,
	)
}

// mergeCheckOnBranchUpdate handles pull request Branch Updated events.
// It updates the PR head git ref to point to the latest commit.
func (s *Service) mergeCheckOnBranchUpdate(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	return s.updateMergeData(
		ctx,
		event.Payload.TargetRepoID,
		event.Payload.Number,
		event.Payload.OldSHA,
		event.Payload.NewSHA,
	)
}

// mergeCheckOnReopen handles pull request StateChanged events.
// It updates the PR head git ref to point to the source branch commit SHA.
func (s *Service) mergeCheckOnReopen(ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload],
) error {
	return s.updateMergeData(
		ctx,
		event.Payload.TargetRepoID,
		event.Payload.Number,
		sha.None.String(),
		event.Payload.SourceSHA,
	)
}

// mergeCheckOnClosed deletes the merge ref.
func (s *Service) mergeCheckOnClosed(ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload],
) error {
	return s.deleteMergeRef(ctx, event.Payload.SourceRepoID, event.Payload.Number)
}

// mergeCheckOnMerged deletes the merge ref.
func (s *Service) mergeCheckOnMerged(ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	return s.deleteMergeRef(ctx, event.Payload.SourceRepoID, event.Payload.Number)
}

func (s *Service) deleteMergeRef(ctx context.Context, repoID int64, prNum int64) error {
	repo, err := s.repoGitInfoCache.Get(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get repo with ID %d: %w", repoID, err)
	}

	writeParams, err := createSystemRPCWriteParams(ctx, s.urlProvider, repo.ID, repo.GitUID)
	if err != nil {
		return fmt.Errorf("failed to generate rpc write params: %w", err)
	}

	// TODO: This doesn't work for forked repos
	err = s.git.UpdateRef(ctx, git.UpdateRefParams{
		WriteParams: writeParams,
		Name:        strconv.Itoa(int(prNum)),
		Type:        gitenum.RefTypePullReqMerge,
		NewValue:    sha.None, // when NewValue is empty will delete the ref.
		OldValue:    sha.None, // we don't care about the old value
	})
	if err != nil {
		return fmt.Errorf("failed to remove PR merge ref: %w", err)
	}

	return nil
}

//nolint:funlen // refactor if required.
func (s *Service) updateMergeData(
	ctx context.Context,
	repoID int64,
	prNum int64,
	oldSHA string,
	newSHA string,
) error {
	pr, err := s.pullreqStore.FindByNumber(ctx, repoID, prNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request number %d: %w", prNum, err)
	}

	// TODO: Merge check should not update the merge base.
	// TODO: Instead it should accept it as an argument and fail if it doesn't match.
	// Then is would not longer be necessary to cancel already active mergeability checks.

	if pr.State != enum.PullReqStateOpen {
		return fmt.Errorf("cannot do mergability check on closed PR %d", pr.Number)
	}

	// cancel all previous mergability work for this PR based on oldSHA
	if err := s.pubsub.Publish(ctx, cancelMergeCheckKey, []byte(oldSHA),
		pubsub.WithPublishNamespace("pullreq")); err != nil {
		return err
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	s.cancelMutex.Lock()
	// NOTE: Temporary workaround to avoid overwriting existing cancel method on same machine.
	// This doesn't avoid same SHA running on multiple machines
	if _, ok := s.cancelMergeability[newSHA]; ok {
		s.cancelMutex.Unlock()
		cancel()
		return nil
	}
	s.cancelMergeability[newSHA] = cancel
	s.cancelMutex.Unlock()

	defer func() {
		cancel()
		s.cancelMutex.Lock()
		delete(s.cancelMergeability, newSHA)
		s.cancelMutex.Unlock()
	}()

	// load repository objects
	targetRepo, err := s.repoGitInfoCache.Get(ctx, pr.TargetRepoID)
	if err != nil {
		return err
	}

	sourceRepo := targetRepo
	if pr.TargetRepoID != pr.SourceRepoID {
		sourceRepo, err = s.repoGitInfoCache.Get(ctx, pr.SourceRepoID)
		if err != nil {
			return err
		}
	}

	writeParams, err := createSystemRPCWriteParams(ctx, s.urlProvider, targetRepo.ID, targetRepo.GitUID)
	if err != nil {
		return fmt.Errorf("failed to generate rpc write params: %w", err)
	}

	// call merge and store output in pr merge reference.
	now := time.Now()
	mergeOutput, err := s.git.Merge(ctx, &git.MergeParams{
		WriteParams:     writeParams,
		BaseBranch:      pr.TargetBranch,
		HeadRepoUID:     sourceRepo.GitUID,
		HeadBranch:      pr.SourceBranch,
		RefType:         gitenum.RefTypePullReqMerge,
		RefName:         strconv.Itoa(int(pr.Number)),
		HeadExpectedSHA: sha.Must(newSHA),
		Force:           true,

		// set committer date to ensure repeatability of merge commit across replicas
		CommitterDate: &now,
	})
	if errors.AsStatus(err) == errors.StatusPreconditionFailed {
		return events.NewDiscardEventErrorf("Source branch %q is not on SHA %q anymore.",
			pr.SourceBranch, newSHA)
	}
	if err != nil {
		return fmt.Errorf("failed to run git merge with base %q and head %q: %w", pr.TargetBranch, pr.SourceBranch, err)
	}

	// Update DB in both cases (failure or success)
	_, err = s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		// to avoid racing conditions with merge
		if pr.State != enum.PullReqStateOpen {
			return errPRNotOpen
		}

		if pr.SourceSHA != newSHA {
			return events.NewDiscardEventErrorf("PR SHA %s is newer than %s", pr.SourceSHA, newSHA)
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
			pr.MergeSHA = ptr.String(mergeOutput.MergeSHA.String())
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
		return fmt.Errorf("failed to update PR merge ref in db with error: %w", err)
	}

	if err = s.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
	}

	return nil
}
