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
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type pullReqWatcherStoreStub struct {
	store.PullReqWatcherStore
	watchers []*types.PullReqWatcher
	err      error
}

func (s *pullReqWatcherStoreStub) List(_ context.Context, _ int64) ([]*types.PullReqWatcher, error) {
	return s.watchers, s.err
}

func watcherOf(p *types.PrincipalInfo) *types.PullReqWatcher {
	return &types.PullReqWatcher{PrincipalID: p.ID, Watcher: *p}
}

func TestListWatchers_FiltersNonUsersSkippedAndUnresolved(t *testing.T) {
	t.Parallel()

	alice := &types.PrincipalInfo{ID: 1, Type: enum.PrincipalTypeUser}
	bob := &types.PrincipalInfo{ID: 2, Type: enum.PrincipalTypeUser}
	sa := &types.PrincipalInfo{ID: 3, Type: enum.PrincipalTypeServiceAccount}
	carol := &types.PrincipalInfo{ID: 4, Type: enum.PrincipalTypeUser}

	svc := &Service{pullReqWatcherStore: &pullReqWatcherStoreStub{watchers: []*types.PullReqWatcher{
		watcherOf(alice), watcherOf(bob), watcherOf(sa), watcherOf(carol),
		{PrincipalID: 5}, // principal info could not be resolved
	}}}

	got, err := svc.listWatchers(context.Background(), 20, bob.ID)
	require.NoError(t, err)
	require.Equal(t, []int64{alice.ID, carol.ID}, recipientIDs(got))
}

func TestListWatchers_StoreError(t *testing.T) {
	t.Parallel()

	svc := &Service{pullReqWatcherStore: &pullReqWatcherStoreStub{err: context.DeadlineExceeded}}

	_, err := svc.listWatchers(context.Background(), 20)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestProcessPullReqStateChangedEvent_IncludesWatchersDeduplicated(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: enum.PrincipalTypeUser}
	modifier := &types.PrincipalInfo{ID: 9, Type: enum.PrincipalTypeUser}
	reviewer := &types.PrincipalInfo{ID: 2, Type: enum.PrincipalTypeUser}
	watcher := &types.PrincipalInfo{ID: 5, Type: enum.PrincipalTypeUser}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{
			author.ID: author, modifier.ID: modifier, reviewer.ID: reviewer, watcher.ID: watcher,
		},
		nil,
		&pullReqReviewerStoreStub{reviewers: []*types.PullReqReviewer{individualReviewer(reviewer)}},
		&userGroupReviewerStoreStub{},
		&userGroupServiceStub{},
	)
	// the reviewer, the author and the state modifier are watching too - only the real watcher is added.
	svc.pullReqWatcherStore = &pullReqWatcherStoreStub{watchers: []*types.PullReqWatcher{
		watcherOf(reviewer), watcherOf(author), watcherOf(modifier), watcherOf(watcher),
	}}

	_, recipients, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateMerged,
	)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{reviewer.ID, author.ID, watcher.ID}, recipientIDs(recipients))
}

func TestProcessCommentCreatedEvent_WatchersJoinParticipantsWithoutDuplicates(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: enum.PrincipalTypeUser}
	commenter := &types.PrincipalInfo{ID: 2, Type: enum.PrincipalTypeUser}
	watcher := &types.PrincipalInfo{ID: 3, Type: enum.PrincipalTypeUser}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      enum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "hello",
	}

	svc := buildFullSvc(author, commenter,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)
	// the commenter and the author watch the PR too - they must not be notified a second time.
	svc.pullReqWatcherStore = &pullReqWatcherStoreStub{watchers: []*types.PullReqWatcher{
		watcherOf(commenter), watcherOf(author), watcherOf(watcher),
	}}

	_, _, participants, gotAuthor, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.NoError(t, err)
	require.Equal(t, []int64{watcher.ID}, recipientIDs(participants))
	require.NotNil(t, gotAuthor)
	require.Equal(t, author.ID, gotAuthor.ID)
}

func TestProcessPullReqBranchUpdatedEvent_WatchersAppendedWithoutDuplicates(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: enum.PrincipalTypeUser}
	committer := &types.PrincipalInfo{ID: 9, Type: enum.PrincipalTypeUser}
	reviewer := &types.PrincipalInfo{ID: 2, Type: enum.PrincipalTypeUser}
	watcher := &types.PrincipalInfo{ID: 5, Type: enum.PrincipalTypeUser}
	serviceAccount := &types.PrincipalInfo{ID: 6, Type: enum.PrincipalTypeServiceAccount}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{
			author.ID: author, committer.ID: committer, reviewer.ID: reviewer,
			watcher.ID: watcher, serviceAccount.ID: serviceAccount,
		},
		nil,
		&pullReqReviewerStoreStub{reviewers: []*types.PullReqReviewer{individualReviewer(reviewer)}},
		&userGroupReviewerStoreStub{},
		&userGroupServiceStub{},
	)
	// the reviewer and the committer watch the PR too, as does a service account.
	svc.pullReqWatcherStore = &pullReqWatcherStoreStub{watchers: []*types.PullReqWatcher{
		watcherOf(reviewer), watcherOf(committer), watcherOf(watcher), watcherOf(serviceAccount),
	}}

	event := &events.Event[*pullreqevents.BranchUpdatedPayload]{
		Payload: &pullreqevents.BranchUpdatedPayload{
			Base:   pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: committer.ID},
			NewSHA: "abc123",
		},
	}

	payload, recipients, err := svc.processPullReqBranchUpdatedEvent(context.Background(), event)
	require.NoError(t, err)
	require.Equal(t, committer.ID, payload.Committer.ID)
	// reviewers keep their position, only the genuine watcher is appended.
	require.Equal(t, []int64{reviewer.ID, watcher.ID}, recipientIDs(recipients))
}

func TestProcessReviewSubmittedEvent_WatchersAppendedAfterAuthor(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: enum.PrincipalTypeUser}
	reviewer := &types.PrincipalInfo{ID: 2, Type: enum.PrincipalTypeUser}
	watcher := &types.PrincipalInfo{ID: 5, Type: enum.PrincipalTypeUser}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{author.ID: author, reviewer.ID: reviewer, watcher.ID: watcher},
		nil,
		&pullReqReviewerStoreStub{},
		&userGroupReviewerStoreStub{},
		&userGroupServiceStub{},
	)
	// the author and the reviewer who submitted the review watch the PR too.
	svc.pullReqWatcherStore = &pullReqWatcherStoreStub{watchers: []*types.PullReqWatcher{
		watcherOf(author), watcherOf(reviewer), watcherOf(watcher),
	}}

	event := &events.Event[*pullreqevents.ReviewSubmittedPayload]{
		Payload: &pullreqevents.ReviewSubmittedPayload{
			Base:       pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerID: reviewer.ID,
			Decision:   enum.PullReqReviewDecisionApproved,
		},
	}

	payload, recipients, err := svc.processReviewSubmittedEvent(context.Background(), event)
	require.NoError(t, err)
	require.Equal(t, reviewer.ID, payload.Reviewer.ID)
	require.Equal(t, []int64{author.ID, watcher.ID}, recipientIDs(recipients))
}

func TestProcessReviewSubmittedEvent_WatcherLookupErrorFails(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: enum.PrincipalTypeUser}
	reviewer := &types.PrincipalInfo{ID: 2, Type: enum.PrincipalTypeUser}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{author.ID: author, reviewer.ID: reviewer},
		nil,
		&pullReqReviewerStoreStub{},
		&userGroupReviewerStoreStub{},
		&userGroupServiceStub{},
	)
	svc.pullReqWatcherStore = &pullReqWatcherStoreStub{err: context.DeadlineExceeded}

	event := &events.Event[*pullreqevents.ReviewSubmittedPayload]{
		Payload: &pullreqevents.ReviewSubmittedPayload{
			Base:       pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ReviewerID: reviewer.ID,
		},
	}

	_, _, err := svc.processReviewSubmittedEvent(context.Background(), event)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
