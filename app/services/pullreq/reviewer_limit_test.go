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
	"errors"
	"strings"
	"testing"

	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
)

func TestCheckReviewerLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		reviewerCount          int64
		reviewerCountErr       error
		userGroupReviewerCount int64
		expectedErr            string
	}{
		{
			name:                   "below limit",
			reviewerCount:          10,
			userGroupReviewerCount: 10,
		},
		{
			name:                   "one below limit",
			reviewerCount:          types.MaxPullReqReviewers - 11,
			userGroupReviewerCount: 10,
		},
		{
			name:          "principals only at limit",
			reviewerCount: types.MaxPullReqReviewers,
			expectedErr:   "can't have more than 200 reviewers",
		},
		{
			name:                   "principals and user groups combined at limit",
			reviewerCount:          types.MaxPullReqReviewers - 10,
			userGroupReviewerCount: 10,
			expectedErr:            "can't have more than 200 reviewers",
		},
		{
			name:             "reviewer count fails",
			reviewerCountErr: errors.New("db down"),
			expectedErr:      "failed to count reviewers",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pullreqStore := &mockstore.PullReqStore{}
			reviewerStore := &mockstore.PullReqReviewerStore{}
			userGroupReviewerStore := &mockstore.UserGroupReviewerStore{}

			pullreqStore.On("FindWithLock", int64(42)).Return(&types.PullReq{ID: 42}, nil).Once()
			reviewerStore.On("Count", int64(42)).Return(test.reviewerCount, test.reviewerCountErr).Once()
			userGroupReviewerStore.On("Count", int64(42)).Return(test.userGroupReviewerCount, nil).Maybe()

			svc := &Service{
				pullreqStore:           pullreqStore,
				reviewerStore:          reviewerStore,
				userGroupReviewerStore: userGroupReviewerStore,
			}

			err := svc.CheckReviewerLimit(context.Background(), 42)
			if test.expectedErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), test.expectedErr) {
				t.Fatalf("unexpected error: %v", err)
			}

			reviewerStore.AssertExpectations(t)
		})
	}
}

func TestAddReviewer_LimitExceeded(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}
	userGroupReviewerStore := &mockstore.UserGroupReviewerStore{}

	reviewerStore.On("Find", int64(42), int64(100)).Return((*types.PullReqReviewer)(nil), store.ErrResourceNotFound).Once()
	pullreqStore.On("FindWithLock", int64(42)).Return(&types.PullReq{ID: 42}, nil).Once()
	reviewerStore.On("Count", int64(42)).Return(int64(types.MaxPullReqReviewers-1), nil).Once()
	userGroupReviewerStore.On("Count", int64(42)).Return(int64(1), nil).Once()

	svc := &Service{
		tx:                      &testTransactor{},
		pullreqStore:            pullreqStore,
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
		userGroupReviewerStore:  userGroupReviewerStore,
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	_, added, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if added {
		t.Fatal("expected added=false")
	}
	if !strings.Contains(err.Error(), "can't have more than 200 reviewers") {
		t.Fatalf("unexpected error: %v", err)
	}

	reviewerStore.AssertExpectations(t)
	userGroupReviewerStore.AssertExpectations(t)
	reviewerStore.AssertNotCalled(t, "Create", mock.Anything)
	suggestionStore.AssertNotCalled(t, "Delete", int64(42), int64(100))
}
