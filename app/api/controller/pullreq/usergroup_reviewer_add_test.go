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
	"testing"

	"github.com/harness/gitness/app/auth/authz/authztest"
	pullreqsvc "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/services/usergroup"
	appstore "github.com/harness/gitness/app/store"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// userGroupStoreStub stubs store.UserGroupStore.Find for UserGroupReviewerAdd.
type userGroupStoreStub struct {
	appstore.UserGroupStore
	userGroup *types.UserGroup
	err       error
}

func (s *userGroupStoreStub) Find(_ context.Context, _ int64) (*types.UserGroup, error) {
	return s.userGroup, s.err
}

// userGroupServiceStub stubs usergroup.Service.ListUserIDsByGroupIDs.
type userGroupServiceStub struct {
	usergroup.Service
	userIDs []int64
	err     error
}

func (s *userGroupServiceStub) ListUserIDsByGroupIDs(_ context.Context, _ []int64) ([]int64, error) {
	return s.userIDs, s.err
}

// TestUserGroupReviewerAdd_CreatesReviewerAndReportsEvent exercises the happy
// path: a new user-group reviewer is persisted inside the tx, a reviewer-added
// event is reported (as "requested"), and a PR activity is written.
func TestUserGroupReviewerAdd_CreatesReviewerAndReportsEvent(t *testing.T) {
	t.Parallel()

	const (
		repoID      int64 = 1
		parentID    int64 = 10
		prID        int64 = 55
		prNum       int64 = 7
		userGroupID int64 = 5
	)

	repo := &types.RepositoryCore{
		ID:       repoID,
		ParentID: parentID,
		Path:     "space/repo",
		State:    enum.RepoStateActive,
	}
	pr := &types.PullReq{ID: prID, Number: prNum, State: enum.PullReqStateOpen}

	pullreqStore := &mockstore.PullReqStore{}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	userGroupReviewerStore := &mockstore.UserGroupReviewerStore{}
	activityStore := &mockstore.PullReqActivityStore{}

	pullreqStore.On("FindByNumber", repoID, prNum).Return(pr, nil).Once()
	reviewerStore.On("List", prID).Return([]*types.PullReqReviewer{}, nil).Once()

	// No existing group reviewer row -> proceed to create a new one.
	userGroupReviewerStore.On("Find", prID, userGroupID).
		Return(nil, basestore.ErrResourceNotFound).Once()

	// CheckReviewerLimit inside the tx.
	pullreqStore.On("FindWithLock", prID).Return(pr, nil).Once()
	reviewerStore.On("Count", prID).Return(int64(0), nil).Once()
	userGroupReviewerStore.On("Count", prID).Return(int64(0), nil).Once()

	userGroupReviewerStore.On("Create", mock.Anything).Return(nil).Once()

	// Post-tx activity write.
	pullreqStore.On("UpdateActivitySeq", mock.Anything).Return(pr, nil).Once()
	activityStore.On("CreateWithPayload", mock.Anything, testSession().Principal.ID, mock.Anything, mock.Anything).
		Return(&types.PullReqActivity{}, nil).Once()

	ctrl := &Controller{
		repoFinder:             testRepoFinder(repo),
		authorizer:             authztest.AllowAuthorizer{},
		tx:                     &txAssertStub{},
		pullreqStore:           pullreqStore,
		reviewerStore:          reviewerStore,
		userGroupReviewerStore: userGroupReviewerStore,
		activityStore:          activityStore,
		userGroupStore:         &userGroupStoreStub{userGroup: &types.UserGroup{ID: userGroupID, Identifier: "grp"}},
		userGroupService:       &userGroupServiceStub{userIDs: []int64{2, 3}},
		pullreqService:         pullreqsvc.NewWithReviewerStores(pullreqStore, reviewerStore, userGroupReviewerStore),
		eventReporter:          mockpullreq.NewStubReporter(t),
	}

	got, err := ctrl.UserGroupReviewerAdd(
		context.Background(), testSession(), "1", prNum,
		&UserGroupReviewerAddInput{UserGroupID: userGroupID},
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, userGroupID, got.UserGroupID)
	require.Equal(t, prID, got.PullReqID)

	pullreqStore.AssertExpectations(t)
	reviewerStore.AssertExpectations(t)
	userGroupReviewerStore.AssertExpectations(t)
	activityStore.AssertExpectations(t)
}
