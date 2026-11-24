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
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"time"
	"unicode/utf8"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	maxReviewerSuggestions = 100
	// MaxSuggestionsPerPullReq is the maximum number of suggestion rows a pull request can hold,
	// and therefore also the maximum number of rows accepted in a single request.
	MaxSuggestionsPerPullReq = 50
	maxSuggestionReasonLen   = 500
	// maxSuggestionCommitSHALen fits a hex sha-1 or sha-256 commit id.
	maxSuggestionCommitSHALen = 64
	maxSuggestionRank         = math.MaxInt32
)

// ReviewerSuggestBatchInput is the request input for suggesting reviewers in batch.
// Exactly one of ReviewerIDs (legacy) or Suggestions must be provided.
type ReviewerSuggestBatchInput struct {
	ReviewerIDs []int64                   `json:"reviewer_ids"`
	Suggestions []ReviewerSuggestionInput `json:"suggestions"`
}

// ReviewerSuggestionInput is a single per-row reviewer suggestion.
type ReviewerSuggestionInput struct {
	PrincipalID int64                       `json:"principal_id"`
	Kind        enum.ReviewerSuggestionKind `json:"kind"`
	Reason      string                      `json:"reason"`
	Rank        *int64                      `json:"rank"`
	UserGroupID *int64                      `json:"user_group_id"`
	CommitSHA   string                      `json:"commit_sha"`
}

// ReviewerSuggestBatchOutput is the response of a batch reviewer suggestion.
type ReviewerSuggestBatchOutput struct {
	Results []ReviewerSuggestResult `json:"results"`
}

// ReviewerSuggestResult is the per-principal outcome of a batch reviewer suggestion.
type ReviewerSuggestResult struct {
	PrincipalID int64                 `json:"principal_id"`
	Status      ReviewerSuggestStatus `json:"status"`
	Error       string                `json:"error,omitempty"`
}

// ReviewerSuggestStatus is the outcome of a single reviewer suggestion in a batch.
type ReviewerSuggestStatus string

const (
	ReviewerSuggestStatusCreated  ReviewerSuggestStatus = "created"
	ReviewerSuggestStatusUpdated  ReviewerSuggestStatus = "updated"
	ReviewerSuggestStatusSkipped  ReviewerSuggestStatus = "skipped"
	ReviewerSuggestStatusRejected ReviewerSuggestStatus = "rejected"
)

func (in *ReviewerSuggestBatchInput) Sanitize() error {
	if len(in.ReviewerIDs) > 0 && len(in.Suggestions) > 0 {
		return errors.InvalidArgument("reviewer_ids and suggestions are mutually exclusive")
	}

	if len(in.Suggestions) > 0 {
		return in.sanitizeSuggestions()
	}

	if len(in.ReviewerIDs) == 0 {
		return errors.InvalidArgument("reviewer_ids must not be empty")
	}

	if len(in.ReviewerIDs) > maxReviewerSuggestions {
		return errors.InvalidArgumentf("reviewer_ids must not exceed %d entries", maxReviewerSuggestions)
	}

	for _, reviewerID := range in.ReviewerIDs {
		if reviewerID <= 0 {
			return errors.InvalidArgument("reviewer_ids must contain only values greater than 0")
		}
	}

	slices.Sort(in.ReviewerIDs)
	in.ReviewerIDs = slices.Compact(in.ReviewerIDs)

	return nil
}

func (in *ReviewerSuggestBatchInput) sanitizeSuggestions() error {
	if len(in.Suggestions) > MaxSuggestionsPerPullReq {
		return errors.InvalidArgumentf("suggestions must not exceed %d entries", MaxSuggestionsPerPullReq)
	}

	for i := range in.Suggestions {
		row := &in.Suggestions[i]

		if row.PrincipalID <= 0 {
			return errors.InvalidArgument("suggestions must contain only principal_id values greater than 0")
		}

		if utf8.RuneCountInString(row.Reason) > maxSuggestionReasonLen {
			return errors.InvalidArgumentf("suggestion reason must not exceed %d characters", maxSuggestionReasonLen)
		}

		if row.Rank != nil && (*row.Rank < 0 || *row.Rank > maxSuggestionRank) {
			return errors.InvalidArgumentf("suggestion rank must be between 0 and %d", maxSuggestionRank)
		}

		if len(row.CommitSHA) > maxSuggestionCommitSHALen {
			return errors.InvalidArgumentf(
				"suggestion commit_sha must not exceed %d characters",
				maxSuggestionCommitSHALen,
			)
		}

		kind, ok := row.Kind.Sanitize()
		if !ok {
			return errors.InvalidArgumentf("suggestion kind %q is not valid", string(row.Kind))
		}
		row.Kind = kind
	}

	// Deduplicate by principal_id keeping the last occurrence, then order deterministically.
	last := make(map[int64]ReviewerSuggestionInput, len(in.Suggestions))
	for _, row := range in.Suggestions {
		last[row.PrincipalID] = row
	}
	rows := make([]ReviewerSuggestionInput, 0, len(last))
	for _, row := range last {
		rows = append(rows, row)
	}
	slices.SortFunc(rows, func(a, b ReviewerSuggestionInput) int {
		return cmp.Compare(a.PrincipalID, b.PrincipalID)
	})
	in.Suggestions = rows

	return nil
}

// AddReviewer validates and adds a reviewer to the pull request.
// It returns the reviewer and whether a new reviewer row was created.
func (s *Service) AddReviewer(
	ctx context.Context,
	principal *types.Principal,
	repo *types.RepositoryCore,
	pr *types.PullReq,
	reviewerID int64,
) (*types.PullReqReviewer, bool, error) {
	if pr.Merged != nil {
		return nil, false, usererror.BadRequest("Can't request review for merged pull request")
	}

	if reviewerID == 0 {
		return nil, false, usererror.BadRequest("Must specify reviewer ID.")
	}

	if reviewerID == pr.CreatedBy {
		return nil, false, usererror.BadRequest("Pull request author can't be added as a reviewer.")
	}

	addedByInfo := principal.ToPrincipalInfo()

	var reviewerType enum.PullReqReviewerType
	switch principal.ID {
	case pr.CreatedBy:
		reviewerType = enum.PullReqReviewerTypeRequested
	case reviewerID:
		reviewerType = enum.PullReqReviewerTypeSelfAssigned
	default:
		reviewerType = enum.PullReqReviewerTypeAssigned
	}

	reviewerInfo := addedByInfo
	if reviewerType != enum.PullReqReviewerTypeSelfAssigned {
		reviewerPrincipal, err := s.principalStore.Find(ctx, reviewerID)
		if err != nil {
			return nil, false, fmt.Errorf("failed to find reviewer principal: %w", err)
		}

		reviewerInfo = reviewerPrincipal.ToPrincipalInfo()

		// TODO: To check the reviewer's access to the repo we create a dummy session object. Fix it.
		if err = apiauth.CheckRepo(ctx, s.authorizer, &auth.Session{
			Principal: *reviewerPrincipal,
			Metadata:  nil,
		}, repo, enum.PermissionRepoReview); err != nil {
			log.Ctx(ctx).Info().Msgf("Reviewer principal: %s access error: %s", reviewerInfo.UID, err)
			return nil, false, usererror.BadRequest("The reviewer doesn't have enough permissions for the repository.")
		}
	}

	var reviewer *types.PullReqReviewer
	added := false

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var findErr error
		reviewer, findErr = s.reviewerStore.Find(ctx, pr.ID, reviewerID)
		if findErr != nil && !errors.Is(findErr, store.ErrResourceNotFound) {
			return findErr
		}

		if reviewer == nil {
			if limitErr := s.CheckReviewerLimit(ctx, pr.ID); limitErr != nil {
				return limitErr
			}

			reviewer = NewPullReqReviewer(
				pr, repo, reviewerInfo, addedByInfo, reviewerType, reviewerID,
			)
			added = true

			if createErr := s.reviewerStore.Create(ctx, reviewer); createErr != nil {
				return createErr
			}
		}

		deleteErr := s.reviewerSuggestionStore.Delete(ctx, pr.ID, reviewerID)
		if errors.Is(deleteErr, store.ErrResourceNotFound) {
			return nil
		}
		if deleteErr != nil {
			return fmt.Errorf("failed to delete reviewer suggestion: %w", deleteErr)
		}

		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to create pull request reviewer: %w", err)
	}

	if !added {
		return reviewer, false, nil
	}

	err = func() error {
		payload := &types.PullRequestActivityPayloadReviewerAdd{
			PrincipalID:  reviewer.PrincipalID,
			ReviewerType: reviewerType,
		}

		metadata := &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{reviewer.PrincipalID}},
		}

		updatedPR, updateErr := s.pullreqStore.UpdateActivitySeq(ctx, pr)
		if updateErr != nil {
			return fmt.Errorf("failed to increment pull request activity sequence: %w", updateErr)
		}

		_, createErr := s.activityStore.CreateWithPayload(ctx, updatedPR, principal.ID, payload, metadata)
		if createErr != nil {
			return fmt.Errorf("failed to create pull request activity: %w", createErr)
		}

		return nil
	}()
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to write pull request activity after adding a reviewer")
	}

	return reviewer, true, nil
}

// NewPullReqReviewer creates new pull request reviewer object.
func NewPullReqReviewer(
	pullReq *types.PullReq,
	repo *types.RepositoryCore,
	reviewerInfo, addedByInfo *types.PrincipalInfo,
	reviewerType enum.PullReqReviewerType,
	reviewerID int64,
) *types.PullReqReviewer {
	now := time.Now().UnixMilli()
	return &types.PullReqReviewer{
		PullReqID:      pullReq.ID,
		PrincipalID:    reviewerID,
		CreatedBy:      addedByInfo.ID,
		Created:        now,
		Updated:        now,
		RepoID:         repo.ID,
		Type:           reviewerType,
		LatestReviewID: nil,
		ReviewDecision: enum.PullReqReviewDecisionPending,
		SHA:            "",
		Reviewer:       *reviewerInfo,
		AddedBy:        *addedByInfo,
	}
}
