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
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

// Map extends principalInfoCacheStub (defined in usergroup_reviewer_added_test.go) with the
// bulk lookup used by processPullReqStateChangedEvent. It mirrors the real cache: if any
// requested id is registered as an error, the whole call fails.
func (s *principalInfoCacheStub) Map(_ context.Context, ids []int64) (map[int64]*types.PrincipalInfo, error) {
	result := make(map[int64]*types.PrincipalInfo, len(ids))
	for _, id := range ids {
		if err, ok := s.errByID[id]; ok {
			return nil, err
		}
		if principal, ok := s.byID[id]; ok {
			result[id] = principal
		}
	}
	return result, nil
}

type pullReqReviewerStoreStub struct {
	store.PullReqReviewerStore
	reviewers []*types.PullReqReviewer
	err       error
}

func (s *pullReqReviewerStoreStub) List(_ context.Context, _ int64) ([]*types.PullReqReviewer, error) {
	return s.reviewers, s.err
}

type userGroupReviewerStoreStub struct {
	store.UserGroupReviewerStore
	reviewers []*types.UserGroupReviewer
	err       error
}

func (s *userGroupReviewerStoreStub) List(_ context.Context, _ int64) ([]*types.UserGroupReviewer, error) {
	return s.reviewers, s.err
}

type userGroupServiceStub struct {
	usergroup.Service
	memberIDs []int64
	err       error
}

func (s *userGroupServiceStub) ListUserIDsByGroupIDs(_ context.Context, _ []int64) ([]int64, error) {
	return s.memberIDs, s.err
}

// recipientIDs is a small helper for order-independent assertions.
func recipientIDs(recipients []*types.PrincipalInfo) []int64 {
	ids := make([]int64, len(recipients))
	for i, r := range recipients {
		ids[i] = r.ID
	}
	return ids
}

// baseStateChangeService wires up a Service with the given stubs and the common
// repo/pullreq/url fixtures shared by every state-change test.
func baseStateChangeService(
	author *types.PrincipalInfo,
	byID map[int64]*types.PrincipalInfo,
	errByID map[int64]error,
	reviewerStore *pullReqReviewerStoreStub,
	userGroupReviewerStore *userGroupReviewerStoreStub,
	userGroupSvc *userGroupServiceStub,
) *Service {
	return &Service{
		repoStore: &repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		pullReqStore: &pullReqStoreStub{pr: &types.PullReq{
			ID:        20,
			Number:    99,
			CreatedBy: author.ID,
			Title:     "PR title",
		}},
		principalInfoCache: &principalInfoCacheStub{
			byID:    byID,
			errByID: errByID,
		},
		pullReqReviewersStore:  reviewerStore,
		userGroupReviewerStore: userGroupReviewerStore,
		userGroupService:       userGroupSvc,
		urlProvider:            &urlProviderStub{prURL: "https://example/pr/99"},
	}
}

func individualReviewer(p *types.PrincipalInfo) *types.PullReqReviewer {
	return &types.PullReqReviewer{PrincipalID: p.ID, Reviewer: *p}
}

func TestProcessPullReqStateChangedEvent_IncludesUserGroupMembers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	modifier := &types.PrincipalInfo{ID: 9, DisplayName: "Modifier", Email: "mod@example.com"}
	reviewer := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	memberBob := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}
	memberCarol := &types.PrincipalInfo{ID: 4, DisplayName: "Carol", Email: "carol@example.com"}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{
			author.ID: author, modifier.ID: modifier,
			reviewer.ID: reviewer, memberBob.ID: memberBob, memberCarol.ID: memberCarol,
		},
		nil,
		&pullReqReviewerStoreStub{reviewers: []*types.PullReqReviewer{individualReviewer(reviewer)}},
		&userGroupReviewerStoreStub{reviewers: []*types.UserGroupReviewer{{UserGroupID: 100}}},
		&userGroupServiceStub{memberIDs: []int64{memberBob.ID, memberCarol.ID}},
	)

	payload, recipients, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateClosed,
	)
	require.NoError(t, err)

	require.Equal(t, PullReqStateClosed, payload.State)
	require.Equal(t, modifier.ID, payload.ChangedBy.ID)
	require.Equal(t, author.ID, payload.Base.Author.ID)

	// Individual reviewer is added first, then group members, then the author.
	require.Len(t, recipients, 4)
	require.Equal(t, reviewer.ID, recipients[0].ID)
	require.ElementsMatch(t,
		[]int64{author.ID, reviewer.ID, memberBob.ID, memberCarol.ID},
		recipientIDs(recipients),
	)
}

func TestProcessPullReqStateChangedEvent_DeduplicatesRecipients(t *testing.T) {
	t.Parallel()

	// author and the individual reviewer are ALSO members of the user group; each must
	// appear exactly once across the merged recipient list.
	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	modifier := &types.PrincipalInfo{ID: 9, DisplayName: "Modifier", Email: "mod@example.com"}
	reviewer := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	memberOnly := &types.PrincipalInfo{ID: 3, DisplayName: "Bob", Email: "bob@example.com"}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{
			author.ID: author, modifier.ID: modifier,
			reviewer.ID: reviewer, memberOnly.ID: memberOnly,
		},
		nil,
		&pullReqReviewerStoreStub{reviewers: []*types.PullReqReviewer{individualReviewer(reviewer)}},
		&userGroupReviewerStoreStub{reviewers: []*types.UserGroupReviewer{{UserGroupID: 100}}},
		&userGroupServiceStub{memberIDs: []int64{author.ID, reviewer.ID, memberOnly.ID}},
	)

	_, recipients, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateReopened,
	)
	require.NoError(t, err)

	require.Len(t, recipients, 3)
	require.ElementsMatch(t,
		[]int64{author.ID, reviewer.ID, memberOnly.ID},
		recipientIDs(recipients),
	)
}

func TestProcessPullReqStateChangedEvent_NoUserGroupReviewers(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	modifier := &types.PrincipalInfo{ID: 9, DisplayName: "Modifier", Email: "mod@example.com"}
	reviewer := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}

	// userGroupService must not be consulted when there are no group reviewers; wiring it to
	// error proves the group branch is skipped.
	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{
			author.ID: author, modifier.ID: modifier, reviewer.ID: reviewer,
		},
		nil,
		&pullReqReviewerStoreStub{reviewers: []*types.PullReqReviewer{individualReviewer(reviewer)}},
		&userGroupReviewerStoreStub{reviewers: nil},
		&userGroupServiceStub{err: errors.New("should not be called")},
	)

	_, recipients, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateMerged,
	)
	require.NoError(t, err)

	require.Len(t, recipients, 2)
	require.ElementsMatch(t, []int64{author.ID, reviewer.ID}, recipientIDs(recipients))
}

func TestProcessPullReqStateChangedEvent_ErrorFromUserGroupReviewerStore(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	modifier := &types.PrincipalInfo{ID: 9, DisplayName: "Modifier", Email: "mod@example.com"}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{author.ID: author, modifier.ID: modifier},
		nil,
		&pullReqReviewerStoreStub{},
		&userGroupReviewerStoreStub{err: errors.New("store failure")},
		&userGroupServiceStub{},
	)

	_, _, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateClosed,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get user group reviewers")
}

func TestProcessPullReqStateChangedEvent_ErrorFromListUserIDsByGroupIDs(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	modifier := &types.PrincipalInfo{ID: 9, DisplayName: "Modifier", Email: "mod@example.com"}

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{author.ID: author, modifier.ID: modifier},
		nil,
		&pullReqReviewerStoreStub{},
		&userGroupReviewerStoreStub{reviewers: []*types.UserGroupReviewer{{UserGroupID: 100}}},
		&userGroupServiceStub{err: errors.New("usergroup api failure")},
	)

	_, _, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateClosed,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to list user group member IDs")
}

func TestProcessPullReqStateChangedEvent_ErrorFromPrincipalInfoCacheMap(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"}
	modifier := &types.PrincipalInfo{ID: 9, DisplayName: "Modifier", Email: "mod@example.com"}
	const memberBobID int64 = 3

	svc := baseStateChangeService(
		author,
		map[int64]*types.PrincipalInfo{author.ID: author, modifier.ID: modifier},
		map[int64]error{memberBobID: errors.New("cache failure")},
		&pullReqReviewerStoreStub{},
		&userGroupReviewerStoreStub{reviewers: []*types.UserGroupReviewer{{UserGroupID: 100}}},
		&userGroupServiceStub{memberIDs: []int64{memberBobID}},
	)

	_, _, err := svc.processPullReqStateChangedEvent(
		context.Background(),
		pullreqevents.Base{PullReqID: 20, TargetRepoID: 10, PrincipalID: modifier.ID},
		PullReqStateClosed,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load principal infos for user group members")
}
