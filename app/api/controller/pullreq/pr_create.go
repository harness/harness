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
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CreateInput struct {
	IsDraft bool `json:"is_draft"`

	Title       string `json:"title"`
	Description string `json:"description"`

	SourceRepoRef string `json:"source_repo_ref"`
	SourceBranch  string `json:"source_branch"`
	TargetBranch  string `json:"target_branch"`

	ReviewerIDs []int64 `json:"reviewer_ids"`
}

func (in *CreateInput) Sanitize() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	if err := validateTitle(in.Title); err != nil {
		return err
	}

	if err := validateDescription(in.Description); err != nil {
		return err
	}

	return nil
}

// Create creates a new pull request.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
) (*types.PullReq, error) {
	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	sourceRepo := targetRepo
	if in.SourceRepoRef != "" {
		sourceRepo, err = c.getRepoCheckAccess(ctx, session, in.SourceRepoRef, enum.PermissionRepoPush)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
	}

	if sourceRepo.ID == targetRepo.ID && in.TargetBranch == in.SourceBranch {
		return nil, usererror.BadRequest("target and source branch can't be the same")
	}

	var sourceSHA sha.SHA

	if sourceSHA, err = c.verifyBranchExistence(ctx, sourceRepo, in.SourceBranch); err != nil {
		return nil, err
	}

	if _, err = c.verifyBranchExistence(ctx, targetRepo, in.TargetBranch); err != nil {
		return nil, err
	}

	if err = c.checkIfAlreadyExists(ctx, targetRepo.ID, sourceRepo.ID, in.TargetBranch, in.SourceBranch); err != nil {
		return nil, err
	}

	mergeBaseResult, err := c.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: git.ReadParams{RepoUID: sourceRepo.GitUID},
		Ref1:       in.SourceBranch,
		Ref2:       in.TargetBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	mergeBaseSHA := mergeBaseResult.MergeBaseSHA

	if mergeBaseSHA == sourceSHA {
		return nil, usererror.BadRequest("The source branch doesn't contain any new commits")
	}

	prStats, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams: git.ReadParams{RepoUID: targetRepo.GitUID},
		BaseRef:    mergeBaseSHA.String(),
		HeadRef:    sourceSHA.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR diff stats: %w", err)
	}

	var pr *types.PullReq
	targetRepoID := targetRepo.ID
	err = controller.TxOptLock(ctx, c.tx, func(ctx context.Context) error {
		if targetRepo == nil {
			targetRepo, err = c.repoStore.Find(ctx, targetRepoID)
			if err != nil {
				return fmt.Errorf("failed to increment pullreq sequence number: %w", err)
			}
		}

		targetRepo.PullReqSeq++
		err = c.repoStore.Update(ctx, targetRepo)
		if err != nil {
			return fmt.Errorf("failed to update pullreq sequence number: %w", err)
		}

		pr = newPullReq(session, targetRepo.PullReqSeq, sourceRepo, targetRepo, in, sourceSHA, mergeBaseSHA)
		pr.Stats = types.PullReqStats{
			DiffStats:       types.NewDiffStats(prStats.Commits, prStats.FilesChanged, prStats.Additions, prStats.Deletions),
			Conversations:   0,
			UnresolvedCount: 0,
		}

		err = c.pullreqStore.Create(ctx, pr)
		if err != nil {
			return fmt.Errorf("pullreq creation failed: %w", err)
		}

		if err := c.createReviewers(ctx, session, in.ReviewerIDs, targetRepo, pr); err != nil {
			return err
		}

		return nil
	}, controller.TxOptionResetFunc(func() {
		targetRepo = nil // on the version conflict error force re-fetch of the target repo
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create pullreq: %w", err)
	}

	c.eventReporter.Created(ctx, &pullreqevents.CreatedPayload{
		Base:         eventBase(pr, &session.Principal),
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
		SourceSHA:    sourceSHA.String(),
	})

	c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeCreatePullRequest,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      sourceRepo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   sourceRepo.ID,
			instrument.PropertyRepositoryName: sourceRepo.Identifier,
			instrument.PropertyPullRequestID:  pr.Number,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create pull request operation: %s", err)
	}

	return pr, nil
}

func (c *Controller) createReviewers(
	ctx context.Context,
	session *auth.Session,
	reviewers []int64,
	repo *types.Repository,
	pr *types.PullReq,
) error {
	if len(reviewers) == 0 {
		return nil
	}

	addedByInfo := session.Principal.ToPrincipalInfo()
	reviewerType := enum.PullReqReviewerTypeRequested

	for _, id := range reviewers {
		if id == addedByInfo.ID {
			return usererror.BadRequest("PR creator cannot be added as a reviewer.")
		}

		reviewerPrincipal, err := c.principalStore.Find(ctx, id)
		if err != nil {
			return usererror.BadRequest("Failed to find principal reviewer.")
		}

		// TODO: To check the reviewer's access to the repo we create a dummy session object. Fix it.
		if err = apiauth.CheckRepo(
			ctx,
			c.authorizer,
			&auth.Session{
				Principal: *reviewerPrincipal,
				Metadata:  nil,
			},
			repo,
			enum.PermissionRepoReview,
		); err != nil {
			if errors.Is(err, apiauth.ErrNotAuthorized) {
				return usererror.BadRequest(
					"The reviewer doesn't have enough permissions for the repository.",
				)
			}
			return fmt.Errorf(
				"reviewer principal: %s access error: %w", reviewerPrincipal.UID, err)
		}

		reviewer := newPullReqReviewer(
			session, pr, repo,
			reviewerPrincipal.ToPrincipalInfo(),
			addedByInfo, reviewerType,
			&ReviewerAddInput{
				ReviewerID: id,
			},
		)

		if err = c.reviewerStore.Create(ctx, reviewer); err != nil {
			return fmt.Errorf("failed to create pull request reviewer: %w", err)
		}
	}

	return nil
}

// newPullReq creates new pull request object.
func newPullReq(
	session *auth.Session,
	number int64,
	sourceRepo *types.Repository,
	targetRepo *types.Repository,
	in *CreateInput,
	sourceSHA, mergeBaseSHA sha.SHA,
) *types.PullReq {
	now := time.Now().UnixMilli()
	return &types.PullReq{
		ID:                0, // the ID will be populated in the data layer
		Version:           0,
		Number:            number,
		CreatedBy:         session.Principal.ID,
		Created:           now,
		Updated:           now,
		Edited:            now,
		State:             enum.PullReqStateOpen,
		IsDraft:           in.IsDraft,
		Title:             in.Title,
		Description:       in.Description,
		SourceRepoID:      sourceRepo.ID,
		SourceBranch:      in.SourceBranch,
		SourceSHA:         sourceSHA.String(),
		TargetRepoID:      targetRepo.ID,
		TargetBranch:      in.TargetBranch,
		ActivitySeq:       0,
		MergedBy:          nil,
		Merged:            nil,
		MergeMethod:       nil,
		MergeBaseSHA:      mergeBaseSHA.String(),
		MergeCheckStatus:  enum.MergeCheckStatusUnchecked,
		RebaseCheckStatus: enum.MergeCheckStatusUnchecked,
		Author:            *session.Principal.ToPrincipalInfo(),
		Merger:            nil,
	}
}
