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
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type repoStoreStub struct {
	store.RepoStore
	repo *types.Repository
	err  error
}

func (s *repoStoreStub) Find(_ context.Context, _ int64) (*types.Repository, error) {
	return s.repo, s.err
}

type pullReqStoreStub struct {
	store.PullReqStore
	pr  *types.PullReq
	err error
}

func (s *pullReqStoreStub) Find(_ context.Context, _ int64) (*types.PullReq, error) {
	return s.pr, s.err
}

type principalInfoCacheStub struct {
	store.PrincipalInfoCache
	byID    map[int64]*types.PrincipalInfo
	errByID map[int64]error
}

func (s *principalInfoCacheStub) Get(_ context.Context, id int64) (*types.PrincipalInfo, error) {
	if err, ok := s.errByID[id]; ok {
		return nil, err
	}
	if principal, ok := s.byID[id]; ok {
		return principal, nil
	}
	return nil, errors.New("principal not found")
}

type urlProviderStub struct {
	url.Provider
	prURL string
}

func (s *urlProviderStub) GenerateUIPRURL(_ context.Context, _ string, _ int64) string {
	return s.prURL
}

func TestProcessUserGroupReviewerAddedEvent_ExcludesAuthorFromMembers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	reviewerB := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}

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
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
			},
		},
		urlProvider: &urlProviderStub{prURL: "https://example/pr/99"},
	}

	event := &events.Event[*pullreqevents.UserGroupReviewerAddedPayload]{
		Payload: &pullreqevents.UserGroupReviewerAddedPayload{
			Base: pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{
				author.ID,
				reviewerA.ID,
				reviewerB.ID,
			},
		},
	}

	base, members, err := svc.processUserGroupReviewerAddedEvent(context.Background(), event)
	require.NoError(t, err)
	require.Equal(t, author.ID, base.Author.ID)
	require.Equal(t, "https://example/pr/99", base.PullReqURL)
	require.Len(t, members, 2)
	require.Equal(t, reviewerA.ID, members[0].ID)
	require.Equal(t, reviewerB.ID, members[1].ID)
}

func TestProcessUserGroupReviewerAddedEvent_ReturnsErrorWhenReviewerLookupFails(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	reviewerB := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}

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
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
			},
			errByID: map[int64]error{
				reviewerB.ID: errors.New("cache failure"),
			},
		},
		urlProvider: &urlProviderStub{prURL: "https://example/pr/99"},
	}

	event := &events.Event[*pullreqevents.UserGroupReviewerAddedPayload]{
		Payload: &pullreqevents.UserGroupReviewerAddedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{reviewerA.ID, reviewerB.ID},
		},
	}

	_, _, err := svc.processUserGroupReviewerAddedEvent(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get reviewer from principalInfoCache")
}

type testNotificationClient struct {
	userGroupReviewerAddedCalls []*userGroupReviewerAddedCall
	reviewerAddedCalls          []*reviewerAddedCall
	userGroupReviewerAddedErr   error
	reviewerAddedErr            error
}

type userGroupReviewerAddedCall struct {
	recipients []*types.PrincipalInfo
	payload    *UserGroupReviewerAddedPayload
}

type reviewerAddedCall struct {
	recipients []*types.PrincipalInfo
	payload    *ReviewerAddedPayload
}

func (c *testNotificationClient) SendUserGroupReviewerAdded(
	_ context.Context, recipients []*types.PrincipalInfo, payload *UserGroupReviewerAddedPayload,
) error {
	c.userGroupReviewerAddedCalls = append(c.userGroupReviewerAddedCalls, &userGroupReviewerAddedCall{
		recipients: recipients,
		payload:    payload,
	})
	return c.userGroupReviewerAddedErr
}

func (c *testNotificationClient) SendReviewerAdded(
	_ context.Context, recipients []*types.PrincipalInfo, payload *ReviewerAddedPayload,
) error {
	c.reviewerAddedCalls = append(c.reviewerAddedCalls, &reviewerAddedCall{
		recipients: recipients,
		payload:    payload,
	})
	return c.reviewerAddedErr
}

func (c *testNotificationClient) SendCommentPRAuthor(
	_ context.Context, _ []*types.PrincipalInfo, _ *CommentPayload) error {
	return nil
}
func (c *testNotificationClient) SendCommentMentions(
	_ context.Context, _ []*types.PrincipalInfo, _ *CommentPayload) error {
	return nil
}
func (c *testNotificationClient) SendCommentParticipants(
	_ context.Context, _ []*types.PrincipalInfo, _ *CommentPayload) error {
	return nil
}
func (c *testNotificationClient) SendPullReqBranchUpdated(
	_ context.Context, _ []*types.PrincipalInfo, _ *PullReqBranchUpdatedPayload) error {
	return nil
}
func (c *testNotificationClient) SendReviewSubmitted(
	_ context.Context, _ []*types.PrincipalInfo, _ *ReviewSubmittedPayload) error {
	return nil
}
func (c *testNotificationClient) SendPullReqStateChanged(
	_ context.Context, _ []*types.PrincipalInfo, _ *PullReqStateChangedPayload) error {
	return nil
}

func TestNotifyUserGroupReviewerAdded_SendsGroupedEmailToAuthorAndIndividualToMembers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	reviewerB := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}

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
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/99"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.UserGroupReviewerAddedPayload]{
		Payload: &pullreqevents.UserGroupReviewerAddedPayload{
			Base: pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{
				author.ID,
				reviewerA.ID,
				reviewerB.ID,
			},
		},
	}

	err := svc.notifyUserGroupReviewerAdded(context.Background(), event)
	require.NoError(t, err)

	// Author should get one email with all reviewers listed
	require.Len(t, notifClient.userGroupReviewerAddedCalls, 1)
	authorCall := notifClient.userGroupReviewerAddedCalls[0]
	require.Len(t, authorCall.recipients, 1)
	require.Equal(t, author.ID, authorCall.recipients[0].ID)
	require.Equal(t, 2, authorCall.payload.ReviewerCount)
	require.Equal(t, "Alice, Bob", authorCall.payload.ReviewerNames)

	// Each reviewer should get an individual email
	require.Len(t, notifClient.reviewerAddedCalls, 2)
	require.Equal(t, reviewerA.ID, notifClient.reviewerAddedCalls[0].recipients[0].ID)
	require.Equal(t, reviewerA.ID, notifClient.reviewerAddedCalls[0].payload.Reviewer.ID)
	require.Equal(t, reviewerB.ID, notifClient.reviewerAddedCalls[1].recipients[0].ID)
	require.Equal(t, reviewerB.ID, notifClient.reviewerAddedCalls[1].payload.Reviewer.ID)
}

func TestNotifyUserGroupReviewerAdded_ReturnsEarlyWhenNoMembers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}

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
				author.ID: author,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/99"},
		notificationClient: notifClient,
	}

	// Only the author is in the group
	event := &events.Event[*pullreqevents.UserGroupReviewerAddedPayload]{
		Payload: &pullreqevents.UserGroupReviewerAddedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{author.ID},
		},
	}

	err := svc.notifyUserGroupReviewerAdded(context.Background(), event)
	require.NoError(t, err)

	// No emails should be sent
	require.Empty(t, notifClient.userGroupReviewerAddedCalls)
	require.Empty(t, notifClient.reviewerAddedCalls)
}

func TestNotifyUserGroupReviewerAdded_ReturnsErrorWhenAuthorEmailFails(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	reviewerA := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}

	notifClient := &testNotificationClient{
		userGroupReviewerAddedErr: errors.New("email service unavailable"),
	}

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
				author.ID:    author,
				reviewerA.ID: reviewerA,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/99"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.UserGroupReviewerAddedPayload]{
		Payload: &pullreqevents.UserGroupReviewerAddedPayload{
			Base:        pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{author.ID, reviewerA.ID},
		},
	}

	err := svc.notifyUserGroupReviewerAdded(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send email")

	// Only author email attempted, no individual reviewer emails sent
	require.Len(t, notifClient.userGroupReviewerAddedCalls, 1)
	require.Empty(t, notifClient.reviewerAddedCalls)
}

func TestNotifyUserGroupReviewerAdded_ReturnsErrorWhenReviewerEmailFails(t *testing.T) {
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
			Number:    99,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &principalInfoCacheStub{
			byID: map[int64]*types.PrincipalInfo{
				author.ID:    author,
				reviewerA.ID: reviewerA,
				reviewerB.ID: reviewerB,
			},
		},
		urlProvider:        &urlProviderStub{prURL: "https://example/pr/99"},
		notificationClient: notifClient,
	}

	event := &events.Event[*pullreqevents.UserGroupReviewerAddedPayload]{
		Payload: &pullreqevents.UserGroupReviewerAddedPayload{
			Base: pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerIDs: []int64{
				author.ID,
				reviewerA.ID,
				reviewerB.ID,
			},
		},
	}

	err := svc.notifyUserGroupReviewerAdded(context.Background(), event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send email")

	// Author email succeeded, but first reviewer email failed
	require.Len(t, notifClient.userGroupReviewerAddedCalls, 1)
	require.Len(t, notifClient.reviewerAddedCalls, 1) // Only first attempt
}
