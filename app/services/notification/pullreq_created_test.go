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

package notification

import (
	"context"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

// prCreatedCacheStub extends principalInfoCacheStub to support Map() method needed for PR created.
type prCreatedCacheStub struct {
	store.PrincipalInfoCache
	byIDMap map[int64]*types.PrincipalInfo
	mapErr  error
}

func (s *prCreatedCacheStub) Get(_ context.Context, id int64) (*types.PrincipalInfo, error) {
	if principal, ok := s.byIDMap[id]; ok {
		return principal, nil
	}
	return nil, errors.New("principal not found")
}

func (s *prCreatedCacheStub) Map(_ context.Context, ids []int64) (map[int64]*types.PrincipalInfo, error) {
	if s.mapErr != nil {
		return nil, s.mapErr
	}
	result := make(map[int64]*types.PrincipalInfo)
	for _, id := range ids {
		if principal, ok := s.byIDMap[id]; ok {
			result[id] = principal
		}
	}
	return result, nil
}

func TestNotifyPullReqCreated_SendsBatchedEmailToAuthorAndIndividualToReviewers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	reviewerB := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}
	reviewerC := &types.PrincipalInfo{ID: 4, DisplayName: "Charlie", Email: "charlie@example.com"}

	notifClient := &testNotificationClient{}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "Add new feature",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
				reviewerC.ID: reviewerC,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{reviewerA.ID, reviewerB.ID, reviewerC.ID},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.NoError(t, err)

	// Author should get ONE batched email with all reviewers listed
	require.Len(t, notifClient.reviewersAddedCalls, 1)
	authorCall := notifClient.reviewersAddedCalls[0]
	require.Len(t, authorCall.recipients, 1)
	require.Equal(t, author.ID, authorCall.recipients[0].ID)
	require.Equal(t, 3, authorCall.payload.ReviewerCount)
	// Check that all names are present (order may vary due to map iteration)
	require.Contains(t, authorCall.payload.ReviewerNames, "Alice")
	require.Contains(t, authorCall.payload.ReviewerNames, "Bob")
	require.Contains(t, authorCall.payload.ReviewerNames, "Charlie")

	// Each reviewer should get an individual email
	require.Len(t, notifClient.reviewerAddedCalls, 3)
	require.Equal(t, reviewerA.ID, notifClient.reviewerAddedCalls[0].recipients[0].ID)
	require.Equal(t, reviewerA.ID, notifClient.reviewerAddedCalls[0].payload.Reviewer.ID)
	require.Equal(t, reviewerB.ID, notifClient.reviewerAddedCalls[1].recipients[0].ID)
	require.Equal(t, reviewerB.ID, notifClient.reviewerAddedCalls[1].payload.Reviewer.ID)
	require.Equal(t, reviewerC.ID, notifClient.reviewerAddedCalls[2].recipients[0].ID)
	require.Equal(t, reviewerC.ID, notifClient.reviewerAddedCalls[2].payload.Reviewer.ID)
}

func TestNotifyPullReqCreated_SingleReviewer(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewer := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}

	notifClient := &testNotificationClient{}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "Fix bug",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID:   author,
				reviewer.ID: reviewer,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{reviewer.ID},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.NoError(t, err)

	// Author gets one email with singular form
	require.Len(t, notifClient.reviewersAddedCalls, 1)
	authorCall := notifClient.reviewersAddedCalls[0]
	require.Equal(t, 1, authorCall.payload.ReviewerCount)
	require.Equal(t, "Alice", authorCall.payload.ReviewerNames)

	// Reviewer gets one email
	require.Len(t, notifClient.reviewerAddedCalls, 1)
	require.Equal(t, reviewer.ID, notifClient.reviewerAddedCalls[0].recipients[0].ID)
}

func TestNotifyPullReqCreated_ReturnsEarlyWhenNoReviewers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}

	notifClient := &testNotificationClient{}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR without reviewers",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID: author,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{}, // No reviewers
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.NoError(t, err)

	// No emails should be sent
	require.Empty(t, notifClient.reviewersAddedCalls)
	require.Empty(t, notifClient.reviewerAddedCalls)
}

func TestNotifyPullReqCreated_ReturnsErrorWhenGetBasePayloadFails(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo not found")

	svc := &Service{
		repoStore:          &repoStoreStub{err: repoErr},
		principalInfoCache: &prCreatedCacheStub{},
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{2, 3},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get base payload")
}

func TestNotifyPullReqCreated_ReturnsErrorWhenPrincipalCacheFails(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	cacheErr := errors.New("cache unavailable")

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID: author,
			},
			mapErr: cacheErr,
		},
		urlProvider: &urlProviderStub{prURL: "https://example/pr/42"},
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{2, 3},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get principal infos from cache")
}

func TestNotifyPullReqCreated_ReturnsErrorWhenAuthorEmailFails(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewer := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}

	notifClient := &testNotificationClient{
		reviewersAddedErr: errors.New("email service unavailable"),
	}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID:   author,
				reviewer.ID: reviewer,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{reviewer.ID},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send batched email to author")

	// Only author email attempted, no individual reviewer emails sent
	require.Len(t, notifClient.reviewersAddedCalls, 1)
	require.Empty(t, notifClient.reviewerAddedCalls)
}

func TestNotifyPullReqCreated_ReturnsErrorWhenReviewerEmailFails(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	reviewerB := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}

	notifClient := &testNotificationClient{
		reviewerAddedErr: errors.New("email service down"),
	}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{reviewerA.ID, reviewerB.ID},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send email to reviewer")

	// Author email succeeded, but first reviewer email failed
	require.Len(t, notifClient.reviewersAddedCalls, 1)
	require.Len(t, notifClient.reviewerAddedCalls, 1) // Only first attempt
}

func TestNotifyPullReqCreated_ExcludesAuthorFromReviewers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	reviewerB := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}

	notifClient := &testNotificationClient{}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	// Author is accidentally included in ReviewerIDs
	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{author.ID, reviewerA.ID, reviewerB.ID},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.NoError(t, err)

	// Author should get ONE email with only the other reviewers listed (not themselves)
	require.Len(t, notifClient.reviewersAddedCalls, 1)
	authorCall := notifClient.reviewersAddedCalls[0]
	require.Equal(t, 2, authorCall.payload.ReviewerCount)
	require.Contains(t, authorCall.payload.ReviewerNames, "Alice")
	require.Contains(t, authorCall.payload.ReviewerNames, "Bob")
	require.NotContains(t, authorCall.payload.ReviewerNames, "Author")

	// Only the actual reviewers get individual emails (not the author)
	require.Len(t, notifClient.reviewerAddedCalls, 2)
	reviewerIDs := []int64{
		notifClient.reviewerAddedCalls[0].recipients[0].ID,
		notifClient.reviewerAddedCalls[1].recipients[0].ID,
	}
	require.Contains(t, reviewerIDs, reviewerA.ID)
	require.Contains(t, reviewerIDs, reviewerB.ID)
	require.NotContains(t, reviewerIDs, author.ID)
}

func TestNotifyPullReqCreated_ReturnsEarlyWhenOnlyAuthorInReviewers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}

	notifClient := &testNotificationClient{}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: map[int64]*types.PrincipalInfo{
				author.ID: author,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	// Only the author is in ReviewerIDs
	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{author.ID},
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.NoError(t, err)

	// No emails should be sent since only the author was in the list
	require.Empty(t, notifClient.reviewersAddedCalls)
	require.Empty(t, notifClient.reviewerAddedCalls)
}

func TestNotifyPullReqCreated_ReviewerNamesAreCommaSeparated(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewers := []*types.PrincipalInfo{
		{ID: 2, DisplayName: "Alice", Email: "alice@example.com"},
		{ID: 3, DisplayName: "Bob", Email: "bob@example.com"},
		{ID: 4, DisplayName: "Charlie", Email: "charlie@example.com"},
		{ID: 5, DisplayName: "Diana", Email: "diana@example.com"},
	}

	notifClient := &testNotificationClient{}

	principalMap := map[int64]*types.PrincipalInfo{author.ID: author}
	var reviewerIDs []int64
	for _, r := range reviewers {
		principalMap[r.ID] = r
		reviewerIDs = append(reviewerIDs, r.ID)
	}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    42,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &prCreatedCacheStub{
			byIDMap: principalMap,
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.CreatedPayload]{
		Payload: &pullreqevents.CreatedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: reviewerIDs,
		},
	}

	err := svc.notifyPullReqCreated(context.Background(), event)
	require.NoError(t, err)

	// Check that author received properly formatted list
	require.Len(t, notifClient.reviewersAddedCalls, 1)
	authorCall := notifClient.reviewersAddedCalls[0]
	require.Equal(t, 4, authorCall.payload.ReviewerCount)
	// Check that all names are present (order may vary due to map iteration)
	require.Contains(t, authorCall.payload.ReviewerNames, "Alice")
	require.Contains(t, authorCall.payload.ReviewerNames, "Bob")
	require.Contains(t, authorCall.payload.ReviewerNames, "Charlie")
	require.Contains(t, authorCall.payload.ReviewerNames, "Diana")
	require.Contains(t, authorCall.payload.ReviewerNames, ", ")
}
