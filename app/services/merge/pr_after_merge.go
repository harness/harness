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
	"time"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

// DatabaseUpdate updates a merged pull request in the database using optimistic locking.
// It also inserts pull request activity and removes auto-merge entry if any.
func (s *Service) DatabaseUpdate(
	ctx context.Context,
	pr *types.PullReq,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	rulesBypassed bool,
	rulesBypassMessage string,
) (prOut *types.PullReq, seqBranchDeleted int64, err error) {
	return s.databaseUpdate(
		ctx,
		pr,
		mergeMethod,
		mergeOutput,
		mergedBy,
		rulesBypassed,
		rulesBypassMessage,
		s.pullreqStore.UpdateOptLock,
	)
}

// DatabaseUpdateNoOptLock updates a merged pull request in the database without optimistic locking mechanism.
// This means that if in the meantime the pull request gets modified,
// the function wouldn't handle the optimistic lock error, but would return it instead.
// It also inserts pull request activity and removes auto-merge entry if any.
func (s *Service) DatabaseUpdateNoOptLock(ctx context.Context,
	pr *types.PullReq,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	rulesBypassed bool,
	rulesBypassMessage string,
) (prOut *types.PullReq, seqBranchDeleted int64, err error) {
	storeFn := func(
		ctx context.Context,
		pr *types.PullReq,
		mutateFn func(*types.PullReq) error,
	) (*types.PullReq, error) {
		if err = mutateFn(pr); err != nil {
			return nil, err
		}
		err = s.pullreqStore.Update(ctx, pr)
		if err != nil {
			return nil, err
		}

		return pr, nil
	}

	return s.databaseUpdate(
		ctx,
		pr,
		mergeMethod,
		mergeOutput,
		mergedBy,
		rulesBypassed,
		rulesBypassMessage,
		storeFn,
	)
}

func (s *Service) databaseUpdate(
	ctx context.Context,
	pr *types.PullReq,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	rulesBypassed bool,
	rulesBypassMessage string,
	storeFn func(context.Context, *types.PullReq, func(pr *types.PullReq) error) (*types.PullReq, error),
) (prOut *types.PullReq, seqBranchDeleted int64, err error) {
	// Mark the PR as merged in the DB.

	var seqMerge int64

	pr, err = storeFn(ctx, pr, func(prToUpdate *types.PullReq) error {
		seqMerge, seqBranchDeleted = s.mutatePullReqAfterMerge(
			prToUpdate,
			mergeMethod,
			mergeOutput,
			mergedBy,
			rulesBypassed,
		)
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to update pull request after merge: %w", err)
	}

	// Insert pull request activity.

	errActMerge := s.InsertActivityMerge(
		ctx,
		pr,
		mergeMethod,
		mergeOutput,
		mergedBy,
		seqMerge,
		rulesBypassed,
		rulesBypassMessage,
	)
	if errActMerge != nil {
		// non-critical error
		log.Ctx(ctx).Err(errActMerge).Msg("failed to write pull req merge activity")
	}

	// Delete the auto-merge entry if exists.

	_, errAutoMerge := s.autoMergeStore.Delete(ctx, pr.ID)
	if errAutoMerge != nil && !errors.Is(errAutoMerge, store.ErrResourceNotFound) {
		// non-critical error
		log.Ctx(ctx).Warn().Err(errAutoMerge).
			Msg("failed to remove auto merge object after merging")
	}

	return pr, seqBranchDeleted, nil
}

// mutatePullReqAfterMerge updates pull request object after merging.
func (s *Service) mutatePullReqAfterMerge(
	pr *types.PullReq,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	rulesBypassed bool,
) (int64, int64) {
	now := time.Now().UnixMilli()
	var activitySeqMerge, activitySeqBranchDeleted int64

	pr.State = enum.PullReqStateMerged
	pr.SubState = enum.PullReqSubStateNone

	pr.Merged = &now
	pr.MergedBy = &mergedBy.ID
	pr.MergeMethod = &mergeMethod

	// update all Merge specific information (might be empty if previous merge check failed)
	// since this is the final operation on the PR, we update any sha that might've changed by now.
	pr.SourceSHA = mergeOutput.HeadSHA.String()
	pr.MergeTargetSHA = ptr.String(mergeOutput.BaseSHA.String())
	pr.MergeBaseSHA = mergeOutput.MergeBaseSHA.String()
	pr.MergeSHA = ptr.String(mergeOutput.MergeSHA.String())
	pr.MarkAsMerged()

	pr.MergeViolationsBypassed = &rulesBypassed

	pr.Stats.DiffStats = types.NewDiffStats(
		mergeOutput.CommitCount,
		mergeOutput.ChangedFileCount,
		mergeOutput.Additions,
		mergeOutput.Deletions,
	)

	// update sequence for PR activities
	pr.ActivitySeq++
	activitySeqMerge = pr.ActivitySeq

	pr.ActivitySeq++
	activitySeqBranchDeleted = pr.ActivitySeq

	return activitySeqMerge, activitySeqBranchDeleted
}

func (s *Service) InsertActivityMerge(
	ctx context.Context,
	pr *types.PullReq,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	activitySeqMerge int64,
	rulesBypassed bool,
	rulesBypassMessage string,
) error {
	pr.ActivitySeq = activitySeqMerge
	activityPayload := &types.PullRequestActivityPayloadMerge{
		MergeMethod:   mergeMethod,
		MergeSHA:      mergeOutput.MergeSHA.String(),
		TargetSHA:     mergeOutput.BaseSHA.String(),
		SourceSHA:     mergeOutput.HeadSHA.String(),
		RulesBypassed: rulesBypassed,
		BypassMessage: rulesBypassMessage,
	}

	_, err := s.activityStore.CreateWithPayload(ctx, pr, mergedBy.ID, activityPayload, nil)

	return err
}

// Publish events about a pull request merging.
func (s *Service) Publish(
	ctx context.Context,
	pr *types.PullReq,
	targetRepo *types.RepositoryCore,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
) {
	s.eventReporter.Merged(ctx, &pullreqevents.MergedPayload{
		Base: pullreqevents.Base{
			PullReqID:    pr.ID,
			SourceRepoID: pr.SourceRepoID,
			TargetRepoID: pr.TargetRepoID,
			Number:       pr.Number,
			PrincipalID:  mergedBy.ID,
		},
		MergeMethod: mergeMethod,
		MergeSHA:    mergeOutput.MergeSHA.String(),
		TargetSHA:   mergeOutput.BaseSHA.String(),
		SourceSHA:   mergeOutput.HeadSHA.String(),
	})

	s.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)

	if err := s.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeMergePullRequest,
		Principal: mergedBy,
		Path:      targetRepo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   targetRepo.ID,
			instrument.PropertyRepositoryName: targetRepo.Identifier,
			instrument.PropertyPullRequestID:  pr.Number,
			instrument.PropertyMergeStrategy:  mergeMethod,
		},
	}); err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for merge pr operation: %s", err)
	}
}

func (s *Service) InsertActivityDeletedBranch(
	ctx context.Context,
	pr *types.PullReq,
	mergedBy *types.PrincipalInfo,
	activitySeqBranchDeleted int64,
) error {
	// NOTE: there is a chance someone pushed on the branch between merge and delete.
	// Either way, we'll use the SHA that was merged with for the activity to be consistent from PR perspective.
	pr.ActivitySeq = activitySeqBranchDeleted
	activityPayload := &types.PullRequestActivityPayloadBranchDelete{SHA: pr.SourceSHA}

	_, err := s.activityStore.CreateWithPayload(ctx, pr, mergedBy.ID, activityPayload, nil)

	return err
}

// DeleteBranchTry attempts to delete the pull request's source branch and insert pull request activity entry.
// It doesn't return an error, only logs the result because in all situations this is the best effort attempt.
func (s *Service) DeleteBranchTry(
	ctx context.Context,
	pr *types.PullReq,
	principalInfo *types.PrincipalInfo,
	seqBranchDeleted int64,
) bool {
	deleted, errDeleted := s.deleteBranch(ctx, pr, principalInfo)
	if errDeleted != nil {
		log.Ctx(ctx).Err(errDeleted).
			Msg("failed to automatically delete pull request source branch")
	}

	if deleted {
		errAct := s.InsertActivityDeletedBranch(ctx, pr, principalInfo, seqBranchDeleted)
		if errAct != nil {
			// non-critical error
			log.Ctx(ctx).Err(errAct).
				Msg("failed to write pull request activity for successful automatic branch delete")
		}
	}

	return deleted
}

func (s *Service) deleteBranch(
	ctx context.Context,
	pr *types.PullReq,
	mergedBy *types.PrincipalInfo,
) (bool, error) {
	if pr.SourceRepoID == nil {
		return false, nil
	}

	sourceWriteParams, err := s.createRPCWriteParams(ctx, mergedBy, *pr.SourceRepoID)
	if err != nil {
		return false, fmt.Errorf("failed to create write params to delete branch: %w", err)
	}

	err = s.git.DeleteBranch(ctx, &git.DeleteBranchParams{
		WriteParams: sourceWriteParams,
		BranchName:  pr.SourceBranch,
	})
	if err != nil {
		return false, fmt.Errorf("failed to delete source branch: %w", err)
	}

	return true, nil
}

func (s *Service) createRPCWriteParams(
	ctx context.Context,
	principal *types.PrincipalInfo,
	repoID int64,
) (git.WriteParams, error) {
	baseURL := s.urlProvider.GetInternalAPIURL(ctx)

	repo, err := s.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return git.WriteParams{}, fmt.Errorf("failed to find repo: %w", err)
	}

	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		baseURL,
		repoID,
		principal.ID,
		false,
		true,
	)
	if err != nil {
		return git.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return git.WriteParams{
		Actor: git.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}
