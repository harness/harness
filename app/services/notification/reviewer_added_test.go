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
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func notifyReviewerAddedForTest(
	t *testing.T,
	author, reviewer *types.PrincipalInfo,
	actorID int64,
) *testNotificationClient {
	t.Helper()

	notifClient := &testNotificationClient{}

	svc := &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    99,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &principalInfoCacheStub{
			byID: map[int64]*types.PrincipalInfo{
				author.ID:   author,
				reviewer.ID: reviewer,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/99"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.ReviewerAddedPayload]{
		Payload: &pullreqevents.ReviewerAddedPayload{
			Base:       pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: actorID},
			ReviewerID: reviewer.ID,
		},
	}

	require.NoError(t, svc.notifyReviewerAdded(context.Background(), event))

	return notifClient
}

func TestNotifyReviewerAdded_SelfAddedReviewerIsNotMailed(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	approver := &types.PrincipalInfo{ID: 2, DisplayName: "Approver", Email: "approver@example.com"}

	// The approver added themselves (approve without being a reviewer, or self-assign).
	notifClient := notifyReviewerAddedForTest(t, author, approver, approver.ID)

	require.Len(t, notifClient.reviewerAddedCalls, 1)
	recipients := notifClient.reviewerAddedCalls[0].recipients
	require.Len(t, recipients, 1)
	require.Equal(t, author.ID, recipients[0].ID)
}

func TestNotifyReviewerAdded_MailsAuthorAndReviewerWhenAddedBySomeoneElse(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewer := &types.PrincipalInfo{ID: 2, DisplayName: "Reviewer", Email: "reviewer@example.com"}

	// The author adds the reviewer.
	notifClient := notifyReviewerAddedForTest(t, author, reviewer, author.ID)

	require.Len(t, notifClient.reviewerAddedCalls, 1)
	recipients := notifClient.reviewerAddedCalls[0].recipients
	require.Len(t, recipients, 2)
	require.Equal(t, author.ID, recipients[0].ID)
	require.Equal(t, reviewer.ID, recipients[1].ID)
}

func TestNotifyReviewerAdded_SendsNothingWhenAuthorSelfAssigns(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}

	notifClient := notifyReviewerAddedForTest(t, author, author, author.ID)

	require.Empty(t, notifClient.reviewerAddedCalls)
}

func TestNotifyReviewerAdded_MailsAuthorOnceWhenSomeoneElseAddsAuthorAsReviewer(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	otherID := int64(2)

	notifClient := notifyReviewerAddedForTest(t, author, author, otherID)

	require.Len(t, notifClient.reviewerAddedCalls, 1)
	recipients := notifClient.reviewerAddedCalls[0].recipients
	require.Len(t, recipients, 1)
	require.Equal(t, author.ID, recipients[0].ID)
}
