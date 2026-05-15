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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	appstore "github.com/harness/gitness/app/store"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
)

type testTransactor struct{}

func (t *testTransactor) WithTx(ctx context.Context, txFn func(ctx context.Context) error, _ ...any) error {
	return txFn(ctx)
}

type testFailingTransactor struct {
	err error
}

func (t *testFailingTransactor) WithTx(_ context.Context, _ func(ctx context.Context) error, _ ...any) error {
	return t.err
}

type testPrincipalStore struct {
	appstore.PrincipalStore
	findFn func(ctx context.Context, id int64) (*types.Principal, error)
}

func (t *testPrincipalStore) Find(ctx context.Context, id int64) (*types.Principal, error) {
	if t.findFn != nil {
		return t.findFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

type denyAuthorizer struct{}

func (d *denyAuthorizer) Check(
	_ context.Context,
	_ *auth.Session,
	_ *types.Scope,
	_ *types.Resource,
	_ enum.Permission,
) (bool, error) {
	return false, nil
}

func (d *denyAuthorizer) CheckAll(
	_ context.Context,
	_ *auth.Session,
	_ ...types.PermissionCheck,
) (bool, error) {
	return false, nil
}

var _ authz.Authorizer = (*denyAuthorizer)(nil)

func TestAddReviewer_Validation(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	ctx := context.Background()
	principal := &types.Principal{ID: 10}
	repo := &types.RepositoryCore{ID: 1}

	t.Run("merged pull request", func(t *testing.T) {
		t.Parallel()

		now := int64(123)
		pr := &types.PullReq{CreatedBy: 11, Merged: &now}

		_, added, err := svc.AddReviewer(ctx, principal, repo, pr, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if added {
			t.Fatal("expected added=false")
		}
		if !strings.Contains(err.Error(), "merged pull request") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing reviewer id", func(t *testing.T) {
		t.Parallel()

		pr := &types.PullReq{CreatedBy: 11}

		_, added, err := svc.AddReviewer(ctx, principal, repo, pr, 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if added {
			t.Fatal("expected added=false")
		}
		if !strings.Contains(err.Error(), "Must specify reviewer ID") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("author cannot be reviewer", func(t *testing.T) {
		t.Parallel()

		pr := &types.PullReq{CreatedBy: 11}

		_, added, err := svc.AddReviewer(ctx, principal, repo, pr, 11)
		if err == nil {
			t.Fatal("expected error")
		}
		if added {
			t.Fatal("expected added=false")
		}
		if !strings.Contains(err.Error(), "author can't be added") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAddReviewer_ExistingReviewerSelfAssigned(t *testing.T) {
	t.Parallel()

	existing := &types.PullReqReviewer{PullReqID: 42, PrincipalID: 100, Type: enum.PullReqReviewerTypeSelfAssigned}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	reviewerStore.On("Find", int64(42), int64(100)).Return(existing, nil).Once()
	suggestionStore.On("Delete", int64(42), int64(100)).Return(nil).Once()

	svc := &Service{
		tx:                      &testTransactor{},
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	reviewer, added, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected existing reviewer, added=false")
	}
	if reviewer != existing {
		t.Fatal("expected existing reviewer pointer")
	}
	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}

func TestAddReviewer_ExistingReviewerDeleteSuggestionError(t *testing.T) {
	t.Parallel()

	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	reviewerStore.
		On("Find", int64(42), int64(100)).
		Return(&types.PullReqReviewer{PullReqID: 42, PrincipalID: 100}, nil).
		Once()
	suggestionStore.On("Delete", int64(42), int64(100)).Return(errors.New("db down")).Once()

	svc := &Service{
		tx:                      &testTransactor{},
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	_, _, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete reviewer suggestion") {
		t.Fatalf("unexpected error: %v", err)
	}
	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}

func TestAddReviewer_FindReviewerError(t *testing.T) {
	t.Parallel()

	reviewerStore := &mockstore.PullReqReviewerStore{}
	reviewerStore.On("Find", int64(42), int64(100)).Return((*types.PullReqReviewer)(nil), errors.New("find failed")).Once()

	svc := &Service{
		tx:                      &testTransactor{},
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: &mockstore.PullReqReviewerSuggestionStore{},
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	_, _, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create pull request reviewer") {
		t.Fatalf("unexpected error: %v", err)
	}
	reviewerStore.AssertExpectations(t)
}

func TestAddReviewer_CreateReviewerError(t *testing.T) {
	t.Parallel()

	reviewerStore := &mockstore.PullReqReviewerStore{}
	reviewerStore.On("Find", int64(42), int64(100)).Return((*types.PullReqReviewer)(nil), store.ErrResourceNotFound).Once()
	reviewerStore.On("Create", mock.AnythingOfType("*types.PullReqReviewer")).Return(errors.New("create failed")).Once()

	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	svc := &Service{
		tx:                      &testTransactor{},
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	_, _, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create pull request reviewer") {
		t.Fatalf("unexpected error: %v", err)
	}

	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertNotCalled(t, "Delete", int64(42), int64(100))
}

func TestAddReviewer_TransactorError(t *testing.T) {
	t.Parallel()

	svc := &Service{
		tx:                      &testFailingTransactor{err: errors.New("tx failed")},
		reviewerStore:           &mockstore.PullReqReviewerStore{},
		reviewerSuggestionStore: &mockstore.PullReqReviewerSuggestionStore{},
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	_, _, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create pull request reviewer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddReviewer_FindReviewerPrincipalError(t *testing.T) {
	t.Parallel()

	principalStore := &testPrincipalStore{
		findFn: func(_ context.Context, id int64) (*types.Principal, error) {
			if id != 200 {
				return nil, errors.New("unexpected id")
			}
			return nil, errors.New("find principal failed")
		},
	}

	svc := &Service{principalStore: principalStore}

	ctx := context.Background()
	principal := &types.Principal{ID: 100, UID: "author"}
	repo := &types.RepositoryCore{ID: 1, Path: "space/repo"}
	pr := &types.PullReq{ID: 42, CreatedBy: 10}

	_, added, err := svc.AddReviewer(ctx, principal, repo, pr, 200)
	if err == nil {
		t.Fatal("expected error")
	}
	if added {
		t.Fatal("expected added=false")
	}
	if !strings.Contains(err.Error(), "failed to find reviewer principal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddReviewer_ReviewerWithoutRepoPermission(t *testing.T) {
	t.Parallel()

	principalStore := &testPrincipalStore{
		findFn: func(_ context.Context, id int64) (*types.Principal, error) {
			if id != 200 {
				return nil, errors.New("unexpected id")
			}
			return &types.Principal{ID: 200, UID: "reviewer", Type: enum.PrincipalTypeUser}, nil
		},
	}

	svc := &Service{
		principalStore: principalStore,
		authorizer:     &denyAuthorizer{},
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100, UID: "author", Type: enum.PrincipalTypeUser}
	repo := &types.RepositoryCore{ID: 1, Path: "space/repo"}
	pr := &types.PullReq{ID: 42, CreatedBy: 10}

	_, added, err := svc.AddReviewer(ctx, principal, repo, pr, 200)
	if err == nil {
		t.Fatal("expected error")
	}
	if added {
		t.Fatal("expected added=false")
	}
	if !strings.Contains(err.Error(), "doesn't have enough permissions") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddReviewer_AddedReviewer_UpdateActivitySeqErrorIgnored(t *testing.T) {
	t.Parallel()

	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}
	pullreqStore := &mockstore.PullReqStore{}
	activityStore := &mockstore.PullReqActivityStore{}

	reviewerStore.On("Find", int64(42), int64(100)).Return((*types.PullReqReviewer)(nil), store.ErrResourceNotFound).Once()
	reviewerStore.On("Create", mock.AnythingOfType("*types.PullReqReviewer")).Return(nil).Once()
	suggestionStore.On("Delete", int64(42), int64(100)).Return(nil).Once()
	pullreqStore.
		On("UpdateActivitySeq", mock.AnythingOfType("*types.PullReq")).
		Return((*types.PullReq)(nil), errors.New("seq failed")).
		Once()

	svc := &Service{
		tx:                      &testTransactor{},
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
		pullreqStore:            pullreqStore,
		activityStore:           activityStore,
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}
	pr := &types.PullReq{ID: 42, CreatedBy: 11}

	reviewer, added, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true")
	}
	if reviewer == nil || reviewer.PrincipalID != 100 {
		t.Fatalf("unexpected reviewer: %#v", reviewer)
	}

	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
	pullreqStore.AssertExpectations(t)
	activityStore.AssertNotCalled(
		t,
		"CreateWithPayload",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestAddReviewer_AddedReviewer_CreateActivityErrorIgnored(t *testing.T) {
	t.Parallel()

	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}
	pullreqStore := &mockstore.PullReqStore{}
	activityStore := &mockstore.PullReqActivityStore{}

	pr := &types.PullReq{ID: 42, CreatedBy: 11}
	updatedPR := &types.PullReq{ID: 42, CreatedBy: 11, ActivitySeq: 10}

	reviewerStore.On("Find", int64(42), int64(100)).Return((*types.PullReqReviewer)(nil), store.ErrResourceNotFound).Once()
	reviewerStore.On("Create", mock.AnythingOfType("*types.PullReqReviewer")).Return(nil).Once()
	suggestionStore.On("Delete", int64(42), int64(100)).Return(nil).Once()
	pullreqStore.On("UpdateActivitySeq", pr).Return(updatedPR, nil).Once()
	activityStore.
		On("CreateWithPayload", updatedPR, int64(100), mock.Anything, mock.Anything).
		Return((*types.PullReqActivity)(nil), errors.New("activity failed")).
		Once()

	svc := &Service{
		tx:                      &testTransactor{},
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
		pullreqStore:            pullreqStore,
		activityStore:           activityStore,
	}

	ctx := context.Background()
	principal := &types.Principal{ID: 100}
	repo := &types.RepositoryCore{ID: 1}

	reviewer, added, err := svc.AddReviewer(ctx, principal, repo, pr, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true")
	}
	if reviewer == nil || reviewer.PrincipalID != 100 {
		t.Fatalf("unexpected reviewer: %#v", reviewer)
	}

	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
	pullreqStore.AssertExpectations(t)
	activityStore.AssertExpectations(t)
}

func TestNewPullReqReviewer(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 21}
	repo := &types.RepositoryCore{ID: 11}
	reviewerInfo := &types.PrincipalInfo{ID: 100, UID: "reviewer"}
	addedByInfo := &types.PrincipalInfo{ID: 200, UID: "author"}

	reviewer := NewPullReqReviewer(
		pr,
		repo,
		reviewerInfo,
		addedByInfo,
		enum.PullReqReviewerTypeAssigned,
		100,
	)

	if reviewer.PullReqID != 21 {
		t.Fatalf("unexpected pull request id: %d", reviewer.PullReqID)
	}
	if reviewer.RepoID != 11 {
		t.Fatalf("unexpected repo id: %d", reviewer.RepoID)
	}
	if reviewer.PrincipalID != 100 {
		t.Fatalf("unexpected reviewer id: %d", reviewer.PrincipalID)
	}
	if reviewer.CreatedBy != 200 {
		t.Fatalf("unexpected created by: %d", reviewer.CreatedBy)
	}
	if reviewer.Type != enum.PullReqReviewerTypeAssigned {
		t.Fatalf("unexpected reviewer type: %s", reviewer.Type)
	}
	if reviewer.ReviewDecision != enum.PullReqReviewDecisionPending {
		t.Fatalf("unexpected review decision: %s", reviewer.ReviewDecision)
	}
	if reviewer.Created <= 0 || reviewer.Updated <= 0 {
		t.Fatalf("expected timestamps to be initialized, got created=%d updated=%d", reviewer.Created, reviewer.Updated)
	}
	if reviewer.Created != reviewer.Updated {
		t.Fatalf("expected created and updated to match, got created=%d updated=%d", reviewer.Created, reviewer.Updated)
	}
}

func TestSanitizeReviewerSuggestBatchInput(t *testing.T) {
	t.Parallel()

	t.Run("empty reviewer IDs", func(t *testing.T) {
		t.Parallel()

		in := &ReviewerSuggestBatchInput{}

		err := in.Sanitize()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "reviewer_ids must not be empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("too many reviewer IDs", func(t *testing.T) {
		t.Parallel()

		reviewerIDs := make([]int64, maxReviewerSuggestions+1)
		for i := range reviewerIDs {
			reviewerIDs[i] = int64(i + 1)
		}

		in := &ReviewerSuggestBatchInput{ReviewerIDs: reviewerIDs}

		err := in.Sanitize()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "reviewer_ids must not exceed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non positive reviewer ID", func(t *testing.T) {
		t.Parallel()

		in := &ReviewerSuggestBatchInput{ReviewerIDs: []int64{1, 0, 2}}

		err := in.Sanitize()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "reviewer_ids must contain only values greater than 0") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("sort and deduplicate", func(t *testing.T) {
		t.Parallel()

		in := &ReviewerSuggestBatchInput{ReviewerIDs: []int64{5, 2, 3, 2, 5, 1}}

		err := in.Sanitize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []int64{1, 2, 3, 5}
		if len(in.ReviewerIDs) != len(want) {
			t.Fatalf("unexpected length: got %d want %d", len(in.ReviewerIDs), len(want))
		}
		for i := range want {
			if in.ReviewerIDs[i] != want[i] {
				t.Fatalf("unexpected reviewer IDs: got %v want %v", in.ReviewerIDs, want)
			}
		}
	})

	t.Run("exact max suggestions allowed", func(t *testing.T) {
		t.Parallel()

		reviewerIDs := make([]int64, maxReviewerSuggestions)
		for i := range reviewerIDs {
			reviewerIDs[i] = int64(maxReviewerSuggestions - i)
		}

		in := &ReviewerSuggestBatchInput{ReviewerIDs: reviewerIDs}

		err := in.Sanitize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(in.ReviewerIDs) != maxReviewerSuggestions {
			t.Fatalf("unexpected length: got %d want %d", len(in.ReviewerIDs), maxReviewerSuggestions)
		}
		if in.ReviewerIDs[0] != 1 || in.ReviewerIDs[len(in.ReviewerIDs)-1] != maxReviewerSuggestions {
			t.Fatalf("expected sorted reviewer IDs, got %v", in.ReviewerIDs)
		}
	})
}
