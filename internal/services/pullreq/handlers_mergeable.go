// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitrpcenum "github.com/harness/gitness/gitrpc/enum"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
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
		event.Payload.Base.PrincipalID,
		event.Payload.TargetRepoID,
		event.Payload.Number,
		gitrpc.NilSHA,
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
		event.Payload.Base.PrincipalID,
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
		event.Payload.Base.PrincipalID,
		event.Payload.TargetRepoID,
		event.Payload.Number,
		"",
		event.Payload.SourceSHA,
	)
}

// mergeCheckOnClosed deletes the merge ref.
func (s *Service) mergeCheckOnClosed(ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload],
) error {
	return s.deleteMergeRef(ctx, event.Payload.PrincipalID, event.Payload.SourceRepoID, event.Payload.PullReqID)
}

// mergeCheckOnMerged deletes the merge ref.
func (s *Service) mergeCheckOnMerged(ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	return s.deleteMergeRef(ctx, event.Payload.PrincipalID, event.Payload.SourceRepoID, event.Payload.PullReqID)
}

func (s *Service) deleteMergeRef(ctx context.Context, principalID int64, repoID int64, prNum int64) error {
	repo, err := s.repoGitInfoCache.Get(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get repo with ID %d: %w", repoID, err)
	}

	principal, err := s.principalCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to find principal with ID %d: %w", principalID, err)
	}

	envars, err := githook.GenerateEnvironmentVariables(&githook.Payload{
		Disabled: true,
	})
	if err != nil {
		return fmt.Errorf("failed to generate githook environment variables: %w", err)
	}

	// TODO: This doesn't work for forked repos
	err = s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
		WriteParams: gitrpc.WriteParams{
			RepoUID: repo.GitUID,
			Actor: gitrpc.Identity{
				Name:  principal.DisplayName,
				Email: principal.Email,
			},
			EnvVars: envars,
		},
		Name:     strconv.Itoa(int(prNum)),
		Type:     gitrpcenum.RefTypePullReqMerge,
		NewValue: "", // when NewValue is empty gitrpc will delete the ref.
		OldValue: "", // we don't care about the old value
	})
	if err != nil {
		return fmt.Errorf("failed to remove PR merge ref: %w", err)
	}

	return nil
}

//nolint:funlen // refactor if required.
func (s *Service) updateMergeData(
	ctx context.Context,
	principalID int64,
	repoID int64,
	prNum int64,
	oldSHA string,
	newSHA string,
) error {
	pr, err := s.pullreqStore.FindByNumber(ctx, repoID, prNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request number %d: %w", prNum, err)
	}

	if pr.State != enum.PullReqStateOpen {
		return fmt.Errorf("cannot do mergability check on closed PR %d", prNum)
	}

	// cancel all previous mergability work for this PR based on oldSHA
	if err = s.pubsub.Publish(ctx, cancelMergeCheckKey, []byte(oldSHA),
		pubsub.WithPublishNamespace("pullreq")); err != nil {
		return err
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	s.cancelMutex.Lock()
	s.cancelMergability[newSHA] = cancel
	s.cancelMutex.Unlock()

	defer func() {
		cancel()
		s.cancelMutex.Lock()
		delete(s.cancelMergability, newSHA)
		s.cancelMutex.Unlock()
	}()

	// load repository objects
	targetRepo, err := s.repoStore.Find(ctx, pr.TargetRepoID)
	if err != nil {
		return err
	}

	sourceRepo := targetRepo
	if pr.TargetRepoID != pr.SourceRepoID {
		sourceRepo, err = s.repoStore.Find(ctx, pr.SourceRepoID)
		if err != nil {
			return err
		}
	}

	principal, err := s.principalCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to find principal with ID %d, error: %w", principalID, err)
	}

	envars, err := githook.GenerateEnvironmentVariables(&githook.Payload{
		Disabled: true,
	})
	if err != nil {
		return fmt.Errorf("failed to generate githook environment variables: %w", err)
	}

	// call merge and store output in pr merge reference.
	var output gitrpc.MergeOutput
	output, err = s.gitRPCClient.Merge(ctx, &gitrpc.MergeParams{
		WriteParams: gitrpc.WriteParams{
			RepoUID: targetRepo.GitUID,
			Actor: gitrpc.Identity{
				Name:  principal.DisplayName,
				Email: principal.Email,
			},
			EnvVars: envars,
		},
		BaseBranch:      pr.TargetBranch,
		HeadRepoUID:     sourceRepo.GitUID,
		HeadBranch:      pr.SourceBranch,
		RefType:         gitrpcenum.RefTypePullReqMerge,
		RefName:         strconv.Itoa(int(prNum)),
		HeadExpectedSHA: newSHA,
		Force:           true,
	})
	if errors.Is(err, gitrpc.ErrPreconditionFailed) {
		// TODO: in case of merge conflict, update conflicts in pr entry - for that we need a
		// nice way to distinguish between the two errors.
		return events.NewDiscardEventErrorf("Source branch '%s' is not on SHA '%s' anymore or is not mergeable.",
			pr.SourceBranch, newSHA)
	}
	if err != nil {
		return fmt.Errorf("merge check failed for %s and %s with err: %w",
			targetRepo.UID+":"+pr.TargetBranch, sourceRepo.UID+":"+pr.SourceBranch, err)
	}

	_, err = s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		if pr.SourceSHA != newSHA {
			return events.NewDiscardEventErrorf("PR SHA %s is newer than %s", pr.SourceSHA, newSHA)
		}
		pr.MergeRefSHA = &output.MergedSHA
		pr.MergeBaseSHA = &output.BaseSHA
		pr.MergeHeadSHA = &output.HeadSHA
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update PR merge ref in db with error: %w", err)
	}

	return nil
}
