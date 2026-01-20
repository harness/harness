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
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

func (s *Service) AfterMerge(
	ctx context.Context,
	pr *types.PullReq,
	targetRepo *types.RepositoryCore,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	rulesBypassed bool,
	rulesBypassMessage string,
	deleteBranch bool,
) (prOut *types.PullReq, branchDeleted bool, err error) {
	// Mark the PR as merged in the DB.

	pr, seqMerge, seqBranchDeleted, err := s.UpdatePullReq(ctx, pr, mergeMethod, mergeOutput, mergedBy, rulesBypassed)
	if err != nil {
		return nil, false, fmt.Errorf("failed to update pull request after merge: %w", err)
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

	// Delete the pull request's source branch.

	if deleteBranch {
		deleted, errDeleted := s.DeleteBranch(ctx, pr, mergedBy)
		if errDeleted != nil {
			log.Ctx(ctx).Err(errDeleted).
				Msg("failed to automaticaly delete pull reuest source branch")
		}

		if deleted {
			errAct := s.InsertActivityDeletedBranch(ctx, pr, mergedBy, seqBranchDeleted)
			if errAct != nil {
				// non-critical error
				log.Ctx(ctx).Err(errAct).
					Msg("failed to write pull request activity for successful automatic branch delete")
			}
		}

		branchDeleted = true
	}

	// Delete the auto-merge entry.

	_, errAutoMerge := s.autoMergeStore.Delete(ctx, pr.ID)
	if errAutoMerge != nil {
		// non-critical error
		log.Ctx(ctx).Err(errAutoMerge).
			Msg("failed to remove auto merge object after merging")
	}

	// Publish events

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

	err = s.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeMergePullRequest,
		Principal: mergedBy,
		Path:      targetRepo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   targetRepo.ID,
			instrument.PropertyRepositoryName: targetRepo.Identifier,
			instrument.PropertyPullRequestID:  pr.Number,
			instrument.PropertyMergeStrategy:  mergeMethod,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for merge pr operation: %s", err)
	}

	return pr, branchDeleted, nil
}

// UpdatePullReq updates pull request in the DB after merging.
func (s *Service) UpdatePullReq(
	ctx context.Context,
	pr *types.PullReq,
	mergeMethod enum.MergeMethod,
	mergeOutput git.MergeOutput,
	mergedBy *types.PrincipalInfo,
	rulesBypassed bool,
) (*types.PullReq, int64, int64, error) {
	now := time.Now().UnixMilli()
	var activitySeqMerge, activitySeqBranchDeleted int64

	pr, err := s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
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

		return nil
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to update pull request: %w", err)
	}

	_, err = s.autoMergeStore.Delete(ctx, pr.ID)
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msg("failed to clear auto merge DB row for the merged pull request")
	}

	return pr, activitySeqMerge, activitySeqBranchDeleted, nil
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

func (s *Service) DeleteBranch(
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
