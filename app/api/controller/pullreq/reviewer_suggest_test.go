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
	pullreqservice "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
)

type allowAuthorizer struct{}

func (a *allowAuthorizer) Check(
	_ context.Context,
	_ *auth.Session,
	_ *types.Scope,
	_ *types.Resource,
	_ enum.Permission,
) (bool, error) {
	return true, nil
}

func (a *allowAuthorizer) CheckAll(
	_ context.Context,
	_ *auth.Session,
	_ ...types.PermissionCheck,
) (bool, error) {
	return true, nil
}

var _ authz.Authorizer = (*allowAuthorizer)(nil)

type repoIDCacheStub struct {
	repo *types.RepositoryCore
}

func (s *repoIDCacheStub) Stats() (int64, int64)            { return 0, 0 }
func (s *repoIDCacheStub) Evict(_ context.Context, _ int64) {}
func (s *repoIDCacheStub) Get(_ context.Context, _ int64) (*types.RepositoryCore, error) {
	if s.repo == nil {
		return nil, errors.New("repo not found")
	}
	return s.repo, nil
}

type spacePathCacheStub struct{}

func (s *spacePathCacheStub) Stats() (int64, int64)             { return 0, 0 }
func (s *spacePathCacheStub) Evict(_ context.Context, _ string) {}
func (s *spacePathCacheStub) Get(_ context.Context, _ string) (*types.SpacePath, error) {
	return nil, errors.New("not used in this test")
}

type repoRefCacheStub struct{}

func (s *repoRefCacheStub) Stats() (int64, int64)                         { return 0, 0 }
func (s *repoRefCacheStub) Evict(_ context.Context, _ types.RepoCacheKey) {}
func (s *repoRefCacheStub) Get(_ context.Context, _ types.RepoCacheKey) (int64, error) {
	return 0, errors.New("not used in this test")
}

func testRepoFinder(repo *types.RepositoryCore) refcache.RepoFinder {
	return refcache.NewRepoFinder(
		nil,
		&spacePathCacheStub{},
		&repoIDCacheStub{repo: repo},
		&repoRefCacheStub{},
		storecache.Evictor[*types.RepositoryCore]{},
	)
}

func testSession() *auth.Session {
	return &auth.Session{Principal: types.Principal{ID: 100, UID: "u-100", Type: enum.PrincipalTypeUser}}
}

func TestReviewerSuggestBatch_InvalidInput(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	session := &auth.Session{}

	err := ctrl.ReviewerSuggestBatch(
		context.Background(),
		session,
		"repo",
		1,
		&pullreqservice.ReviewerSuggestBatchInput{ReviewerIDs: []int64{}},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Invalid reviewer IDs") {
		t.Fatalf("expected invalid reviewer IDs error, got: %v", err)
	}
}

func TestReviewerSuggestApply_EmptyRepoRef(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	session := &auth.Session{}

	_, err := ctrl.ReviewerSuggestApply(context.Background(), session, "", 1, 2)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "A valid repository reference must be provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewerSuggestDelete_NotFound(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return(&types.PullReq{ID: 55, Number: 7}, nil).Once()
	suggestionStore.On("Delete", int64(55), int64(123)).Return(basestore.ErrResourceNotFound).Once()

	ctrl := &Controller{
		authorizer:              &allowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerSuggestionStore: suggestionStore,
	}

	err := ctrl.ReviewerSuggestDelete(context.Background(), testSession(), "1", 7, 123)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "could not be found") {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}

func TestReviewerSuggestBatch_AuthorCannotBeSuggested(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	pullreqStore.
		On("FindByNumber", int64(1), int64(7)).
		Return(&types.PullReq{ID: 55, Number: 7, CreatedBy: 101}, nil).
		Once()
	reviewerStore.On("List", int64(55)).Return([]*types.PullReqReviewer{}, nil).Once()

	ctrl := &Controller{
		authorizer:              &allowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	err := ctrl.ReviewerSuggestBatch(
		context.Background(),
		&auth.Session{Principal: types.Principal{ID: 100, UID: "u-100", Type: enum.PrincipalTypeUser}},
		"1",
		7,
		&pullreqservice.ReviewerSuggestBatchInput{ReviewerIDs: []int64{101}},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "can't be suggested as a reviewer") {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertNotCalled(t, "CreateMany", mock.Anything)
}

func TestReviewerSuggestBatch_AllAlreadyReviewersStillCreateMany(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).
		Return(&types.PullReq{ID: 55, Number: 7, CreatedBy: 101}, nil).Once()
	reviewerStore.On("List", int64(55)).
		Return([]*types.PullReqReviewer{{PrincipalID: 200}}, nil).Once()
	suggestionStore.On("CreateMany", mock.AnythingOfType("[]*types.PullReqReviewerSuggestion")).Return(nil).Once()

	ctrl := &Controller{
		authorizer:              &allowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	err := ctrl.ReviewerSuggestBatch(
		context.Background(),
		&auth.Session{Principal: types.Principal{ID: 100, UID: "u-100", Type: enum.PrincipalTypeUser}},
		"1",
		7,
		&pullreqservice.ReviewerSuggestBatchInput{ReviewerIDs: []int64{200}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}

func TestReviewerSuggestApply_NotFoundSuggestion(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return(&types.PullReq{ID: 55, Number: 7}, nil).Once()
	suggestionStore.
		On("Find", int64(55), int64(123)).
		Return((*types.PullReqReviewerSuggestion)(nil), basestore.ErrResourceNotFound).
		Once()

	ctrl := &Controller{
		authorizer:              &allowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerSuggestionStore: suggestionStore,
	}

	_, err := ctrl.ReviewerSuggestApply(context.Background(), testSession(), "1", 7, 123)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "could not be found") {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}

func TestReviewerSuggestApply_FindSuggestionError(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return(&types.PullReq{ID: 55, Number: 7}, nil).Once()
	suggestionStore.
		On("Find", int64(55), int64(123)).
		Return((*types.PullReqReviewerSuggestion)(nil), errors.New("db down")).
		Once()

	ctrl := &Controller{
		authorizer:              &allowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerSuggestionStore: suggestionStore,
	}

	_, err := ctrl.ReviewerSuggestApply(context.Background(), testSession(), "1", 7, 123)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to find reviewer suggestion") {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}

func TestReviewerSuggestDelete_EmptyRepoRef(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	session := &auth.Session{}

	err := ctrl.ReviewerSuggestDelete(context.Background(), session, "", 1, 2)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "A valid repository reference must be provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListSuggestedReviewers_EmptyRepoRef(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	session := &auth.Session{}

	_, _, err := ctrl.ListSuggestedReviewers(context.Background(), session, "", 1, types.Pagination{Page: 1, Size: 10})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "A valid repository reference must be provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}
