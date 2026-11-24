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
	"slices"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	pullreqservice "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ReviewerSuggestBatch stores reviewer suggestions for a pull request.
// A legacy reviewer_ids request keeps its all-or-nothing behaviour and returns a nil output,
// a suggestions request reports one result per requested row.
func (c *Controller) ReviewerSuggestBatch(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *pullreqservice.ReviewerSuggestBatchInput,
) (*pullreqservice.ReviewerSuggestBatchOutput, error) {
	if err := in.Sanitize(); err != nil {
		// The legacy form keeps its wording, the per-row form reports the validation error as is.
		if len(in.Suggestions) > 0 {
			return nil, usererror.BadRequest(err.Error())
		}
		return nil, usererror.BadRequestf("Invalid reviewer IDs: %v", err)
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	reviewers, err := c.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull request reviewers: %w", err)
	}

	existingReviewerIDs := make(map[int64]struct{}, len(reviewers))
	for _, reviewer := range reviewers {
		existingReviewerIDs[reviewer.PrincipalID] = struct{}{}
	}

	if len(in.Suggestions) > 0 {
		return c.suggestReviewersPerRow(ctx, session, repo, pr, in.Suggestions, existingReviewerIDs)
	}

	// The legacy path keeps its original behaviour: no per-row results, no reason or kind and
	// deliberately no per-pull-request cap, which only applies to the suggestions form.
	suggestions := make([]*types.PullReqReviewerSuggestion, 0, len(in.ReviewerIDs))
	for _, reviewerID := range in.ReviewerIDs {
		if _, ok := existingReviewerIDs[reviewerID]; ok {
			continue
		}

		if reviewerID == pr.CreatedBy {
			return nil, usererror.BadRequestf("Pull request author with ID %d can't be suggested as a reviewer.", reviewerID)
		}

		reviewerPrincipal, err := c.principalStore.Find(ctx, reviewerID)
		if errors.Is(err, store.ErrResourceNotFound) {
			return nil, usererror.NotFoundf("Suggested reviewer with ID %d could not be found.", reviewerID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to find reviewer principal: %w", err)
		}

		err = apiauth.CheckRepo(ctx, c.authorizer, &auth.Session{
			Principal: *reviewerPrincipal,
			Metadata:  nil,
		}, repo, enum.PermissionRepoReview)
		if err != nil {
			if !apiauth.IsNoAccess(err) {
				return nil, fmt.Errorf("failed to check suggested reviewer access: %w", err)
			}
			return nil, usererror.BadRequestf(
				"Suggested reviewer with ID %d does not have permission to review this repository.",
				reviewerID,
			)
		}

		suggestions = append(suggestions, &types.PullReqReviewerSuggestion{
			PullReqID:   pr.ID,
			CreatedBy:   session.Principal.ID,
			PrincipalID: reviewerID,
		})
	}

	if len(suggestions) == 0 {
		return nil, nil //nolint:nilnil // the legacy reviewer_ids path has no output.
	}

	err = c.reviewerSuggestionStore.CreateMany(ctx, suggestions)
	if err != nil {
		return nil, fmt.Errorf("failed to create reviewer suggestions: %w", err)
	}

	return nil, nil //nolint:nilnil // the legacy reviewer_ids path has no output.
}

// suggestReviewersPerRow validates every requested suggestion on its own and reports its outcome,
// so that a single bad row doesn't discard the good ones.
func (c *Controller) suggestReviewersPerRow(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
	pr *types.PullReq,
	rows []pullreqservice.ReviewerSuggestionInput,
	existingReviewerIDs map[int64]struct{},
) (*pullreqservice.ReviewerSuggestBatchOutput, error) {
	results := make([]pullreqservice.ReviewerSuggestResult, len(rows))
	for i := range rows {
		results[i] = pullreqservice.ReviewerSuggestResult{PrincipalID: rows[i].PrincipalID}
	}

	reject := func(i int, format string, args ...any) {
		results[i].Status = pullreqservice.ReviewerSuggestStatusRejected
		results[i].Error = fmt.Sprintf(format, args...)
	}

	// The user groups and the repository's space hierarchy are only looked up once a row
	// reaches the user group check.
	var userGroups map[int64]*types.UserGroup
	var spaceIDs map[int64]struct{}

	candidates := make([]int, 0, len(rows))
	for i := range rows {
		row := rows[i]

		if _, ok := existingReviewerIDs[row.PrincipalID]; ok {
			results[i].Status = pullreqservice.ReviewerSuggestStatusSkipped
			continue
		}

		if row.PrincipalID == pr.CreatedBy {
			reject(i, "Pull request author with ID %d can't be suggested as a reviewer.", row.PrincipalID)
			continue
		}

		principal, findErr := c.principalStore.Find(ctx, row.PrincipalID)
		if errors.Is(findErr, store.ErrResourceNotFound) {
			reject(i, "Suggested reviewer with ID %d could not be found.", row.PrincipalID)
			continue
		}
		if findErr != nil {
			return nil, fmt.Errorf("failed to find reviewer principal: %w", findErr)
		}

		// TODO: To check the reviewer's access to the repo we create a dummy session object. Fix it.
		accessErr := apiauth.CheckRepo(ctx, c.authorizer, &auth.Session{
			Principal: *principal,
			Metadata:  nil,
		}, repo, enum.PermissionRepoReview)
		if accessErr != nil {
			if !apiauth.IsNoAccess(accessErr) {
				return nil, fmt.Errorf("failed to check suggested reviewer access: %w", accessErr)
			}
			reject(
				i,
				"Suggested reviewer with ID %d does not have permission to review this repository.",
				row.PrincipalID,
			)
			continue
		}

		if row.UserGroupID != nil && userGroups == nil {
			var mapErr error
			userGroups, mapErr = c.mapSuggestionUserGroups(ctx, rows)
			if mapErr != nil {
				return nil, mapErr
			}

			spaceIDs, mapErr = c.repoSpaceHierarchyIDs(ctx, repo.ParentID)
			if mapErr != nil {
				return nil, mapErr
			}
		}

		if msg := checkSuggestionUserGroup(row.UserGroupID, userGroups, spaceIDs); msg != "" {
			reject(i, "%s", msg)
			continue
		}

		candidates = append(candidates, i)
	}

	if len(candidates) == 0 {
		return &pullreqservice.ReviewerSuggestBatchOutput{Results: results}, nil
	}

	// The cap check, the existing row reads and the write all run in one serializable transaction,
	// so concurrent batches can't both pass the cap. WithTx doesn't retry: a serialization failure
	// is returned as an error and the caller may retry the request.
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		count, countErr := c.reviewerSuggestionStore.Count(ctx, pr.ID)
		if countErr != nil {
			return fmt.Errorf("failed to count reviewer suggestions: %w", countErr)
		}

		// A legacy pull request can hold more rows than the cap, so every principal is
		// resolved on its own instead of listing a capped page.
		existing := make(map[int64]*types.PullReqReviewerSuggestion, len(candidates))
		for _, i := range candidates {
			found, findErr := c.reviewerSuggestionStore.Find(ctx, pr.ID, rows[i].PrincipalID)
			if findErr != nil && !errors.Is(findErr, store.ErrResourceNotFound) {
				return fmt.Errorf("failed to find reviewer suggestion: %w", findErr)
			}
			if found != nil {
				existing[rows[i].PrincipalID] = found
			}
		}

		// Only a service may overwrite what a service stored, so the creators of the existing
		// rows are resolved in one go, and only when the caller isn't a service itself.
		var serviceCreated map[int64]bool
		if !canSetSuggestionDetail(session) && len(existing) > 0 {
			var ownerErr error
			serviceCreated, ownerErr = c.serviceCreatedSuggestions(ctx, existing)
			if ownerErr != nil {
				return ownerErr
			}
		}

		toUpsert := make([]*types.PullReqReviewerSuggestion, 0, len(candidates))
		for _, i := range candidates {
			row := rows[i]

			status := pullreqservice.ReviewerSuggestStatusUpdated
			if _, ok := existing[row.PrincipalID]; ok {
				if serviceCreated[row.PrincipalID] {
					reject(
						i,
						"Suggestion for reviewer with ID %d was created by a service account and can't be overwritten.",
						row.PrincipalID,
					)
					continue
				}
			} else {
				// Refreshing an existing principal doesn't count toward the cap.
				if count >= pullreqservice.MaxSuggestionsPerPullReq {
					reject(
						i,
						"Pull request already holds the maximum of %d reviewer suggestions.",
						pullreqservice.MaxSuggestionsPerPullReq,
					)
					continue
				}
				count++
				status = pullreqservice.ReviewerSuggestStatusCreated
			}

			toUpsert = append(toUpsert, newReviewerSuggestion(pr, session, row, existing[row.PrincipalID]))
			results[i].Status = status
		}

		if len(toUpsert) == 0 {
			return nil
		}

		if upsertErr := c.reviewerSuggestionStore.UpsertMany(ctx, toUpsert); upsertErr != nil {
			return fmt.Errorf("failed to upsert reviewer suggestions: %w", upsertErr)
		}

		return nil
	}, dbtx.TxSerializable)
	if err != nil {
		return nil, err
	}

	return &pullreqservice.ReviewerSuggestBatchOutput{Results: results}, nil
}

// mapSuggestionUserGroups resolves the distinct user groups referenced by the requested rows.
// The returned map is never nil, so the caller can use it as the "already looked up" marker.
// None of the referenced groups existing is not an error, the rows referencing them are rejected.
func (c *Controller) mapSuggestionUserGroups(
	ctx context.Context,
	rows []pullreqservice.ReviewerSuggestionInput,
) (map[int64]*types.UserGroup, error) {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.UserGroupID != nil {
			ids = append(ids, *row.UserGroupID)
		}
	}
	if len(ids) == 0 {
		return map[int64]*types.UserGroup{}, nil
	}

	slices.Sort(ids)
	ids = slices.Compact(ids)

	userGroups, err := c.userGroupStore.Map(ctx, ids)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to map suggestion user groups: %w", err)
	}
	if userGroups == nil {
		userGroups = map[int64]*types.UserGroup{}
	}

	return userGroups, nil
}

// repoSpaceHierarchyIDs returns the repository's parent space and all of its ancestors.
// Harness syncs user groups into the account, the org and the project space, so a suggestion
// may legitimately reference a group that lives in any space above the repository.
func (c *Controller) repoSpaceHierarchyIDs(ctx context.Context, parentID int64) (map[int64]struct{}, error) {
	ancestorIDs, err := c.spaceStore.GetAncestorIDs(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ancestor ids of the repository's space: %w", err)
	}

	spaceIDs := make(map[int64]struct{}, len(ancestorIDs)+1)
	spaceIDs[parentID] = struct{}{}
	for _, id := range ancestorIDs {
		spaceIDs[id] = struct{}{}
	}

	return spaceIDs, nil
}

// checkSuggestionUserGroup returns the rejection reason if the referenced user group doesn't
// exist in the repository's space hierarchy, and an empty string if the reference is fine.
func checkSuggestionUserGroup(
	userGroupID *int64,
	userGroups map[int64]*types.UserGroup,
	spaceIDs map[int64]struct{},
) string {
	if userGroupID == nil {
		return ""
	}

	userGroup, ok := userGroups[*userGroupID]
	if !ok {
		return fmt.Sprintf("Suggested user group with ID %d could not be found.", *userGroupID)
	}

	if _, ok := spaceIDs[userGroup.SpaceID]; !ok {
		return fmt.Sprintf(
			"Suggested user group with ID %d does not belong to the repository's space hierarchy.",
			*userGroupID,
		)
	}

	return ""
}

// canSetSuggestionDetail reports whether the caller may set the kind and reason of a suggestion.
func canSetSuggestionDetail(session *auth.Session) bool {
	return isServicePrincipal(session.Principal.Type)
}

// isServicePrincipal reports whether the principal type belongs to a service, not a human.
func isServicePrincipal(principalType enum.PrincipalType) bool {
	return principalType == enum.PrincipalTypeService ||
		principalType == enum.PrincipalTypeServiceAccount
}

// serviceCreatedSuggestions reports, per suggested principal, whether the stored row was created
// by a service. Ownership follows the row's creator and not its content, because created_by is
// preserved across upserts while the content a user may legitimately set isn't a reliable marker.
func (c *Controller) serviceCreatedSuggestions(
	ctx context.Context,
	existing map[int64]*types.PullReqReviewerSuggestion,
) (map[int64]bool, error) {
	creatorIDs := make([]int64, 0, len(existing))
	for _, suggestion := range existing {
		creatorIDs = append(creatorIDs, suggestion.CreatedBy)
	}
	slices.Sort(creatorIDs)
	creatorIDs = slices.Compact(creatorIDs)

	creators, err := c.principalInfoCache.Map(ctx, creatorIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to map reviewer suggestion creators: %w", err)
	}

	serviceCreated := make(map[int64]bool, len(existing))
	for principalID, suggestion := range existing {
		// A creator that no longer resolves is treated as a user, so the row stays editable.
		creator, ok := creators[suggestion.CreatedBy]
		serviceCreated[principalID] = ok && isServicePrincipal(creator.Type)
	}

	return serviceCreated, nil
}

// newReviewerSuggestion builds the row to store. A user caller can't set kind and reason, so a
// refresh keeps the ones already stored (a service may have enriched a row the user created).
func newReviewerSuggestion(
	pr *types.PullReq,
	session *auth.Session,
	row pullreqservice.ReviewerSuggestionInput,
	existing *types.PullReqReviewerSuggestion,
) *types.PullReqReviewerSuggestion {
	suggestion := &types.PullReqReviewerSuggestion{
		PullReqID:   pr.ID,
		CreatedBy:   session.Principal.ID,
		PrincipalID: row.PrincipalID,
		Kind:        enum.ReviewerSuggestionKindOptional,
		Rank:        row.Rank,
		UserGroupID: row.UserGroupID,
		CommitSHA:   row.CommitSHA,
	}

	if canSetSuggestionDetail(session) {
		suggestion.Reason = row.Reason
		if row.Kind != "" {
			suggestion.Kind = row.Kind
		}
	} else if existing != nil {
		suggestion.Reason = existing.Reason
		suggestion.Kind = existing.Kind
	}

	return suggestion
}
