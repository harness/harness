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
	"fmt"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/auth/authz/authztest"
	pullreqservice "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/services/refcache"
	appstore "github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func testServiceSession() *auth.Session {
	return &auth.Session{Principal: types.Principal{ID: 300, UID: "svc-300", Type: enum.PrincipalTypeService}}
}

// passthroughTx is a dbtx.Transactor that runs the transaction function directly and records the
// options it was given, standing in for a real database transaction.
type passthroughTx struct {
	opts []any
}

func (t *passthroughTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, opts ...any) error {
	t.opts = opts
	return fn(ctx)
}

// principalStoreStub resolves only the principals it was given, every other ID is unknown.
type principalStoreStub struct {
	appstore.PrincipalStore
	principals map[int64]*types.Principal
}

func (s principalStoreStub) Find(_ context.Context, id int64) (*types.Principal, error) {
	principal, ok := s.principals[id]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}
	return principal, nil
}

// principalInfoCacheStub maps only the principals it was given, mirroring the real cache which
// silently omits IDs it can't resolve.
type principalInfoCacheStub struct {
	appstore.PrincipalInfoCache
	principals map[int64]*types.Principal
}

func (s principalInfoCacheStub) Map(_ context.Context, ids []int64) (map[int64]*types.PrincipalInfo, error) {
	infos := make(map[int64]*types.PrincipalInfo, len(ids))
	for _, id := range ids {
		if principal, ok := s.principals[id]; ok {
			infos[id] = principal.ToPrincipalInfo()
		}
	}
	return infos, nil
}

// spaceStoreStub answers the ancestor lookup of the repository's parent space.
type spaceStoreStub struct {
	appstore.SpaceStore
	ancestorIDs []int64
}

func (s spaceStoreStub) GetAncestorIDs(_ context.Context, spaceID int64) ([]int64, error) {
	return append([]int64{spaceID}, s.ancestorIDs...), nil
}

// principalDenyAuthorizer grants every permission check except the ones for the listed principals.
type principalDenyAuthorizer struct {
	denied map[int64]struct{}
}

func (a principalDenyAuthorizer) Check(
	_ context.Context, session *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	_, denied := a.denied[session.Principal.ID]
	return !denied, nil
}

func (a principalDenyAuthorizer) CheckAll(
	_ context.Context, session *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	_, denied := a.denied[session.Principal.ID]
	return !denied, nil
}

func (a principalDenyAuthorizer) CheckMany(
	_ context.Context, session *auth.Session, permissionChecks ...types.PermissionCheck,
) ([]bool, error) {
	_, denied := a.denied[session.Principal.ID]
	results := make([]bool, len(permissionChecks))
	for i := range results {
		results[i] = !denied
	}
	return results, nil
}

// suggestBatchFixture wires the dependencies ReviewerSuggestBatch uses and pre-registers the
// pull request and reviewer lookups every call makes.
type suggestBatchFixture struct {
	repo            *types.RepositoryCore
	prID            int64
	tx              *passthroughTx
	pullreqStore    *mockstore.PullReqStore
	reviewerStore   *mockstore.PullReqReviewerStore
	suggestionStore *mockstore.PullReqReviewerSuggestionStore
	authorizer      authz.Authorizer
	principals      map[int64]*types.Principal
	groups          map[int64]*types.UserGroup
	// ancestorIDs are the spaces above the repository's parent space.
	ancestorIDs []int64
}

func newSuggestBatchFixture(pr *types.PullReq, reviewers ...*types.PullReqReviewer) *suggestBatchFixture {
	f := &suggestBatchFixture{
		repo:            &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive},
		prID:            pr.ID,
		tx:              &passthroughTx{},
		pullreqStore:    &mockstore.PullReqStore{},
		reviewerStore:   &mockstore.PullReqReviewerStore{},
		suggestionStore: &mockstore.PullReqReviewerSuggestionStore{},
		authorizer:      authztest.AllowAuthorizer{},
		principals:      map[int64]*types.Principal{},
		groups:          map[int64]*types.UserGroup{},
		// The repo lives in space 10, which sits under the org 5 and the account 1.
		ancestorIDs: []int64{5, 1},
	}

	f.pullreqStore.On("FindByNumber", int64(1), pr.Number).Return(pr, nil).Once()
	f.reviewerStore.On("List", pr.ID).Return(reviewers, nil).Once()

	return f
}

func (f *suggestBatchFixture) controller() *Controller {
	return &Controller{
		tx:                      f.tx,
		authorizer:              f.authorizer,
		repoFinder:              testRepoFinder(f.repo),
		pullreqStore:            f.pullreqStore,
		reviewerStore:           f.reviewerStore,
		reviewerSuggestionStore: f.suggestionStore,
		principalStore:          principalStoreStub{principals: f.principals},
		principalInfoCache:      principalInfoCacheStub{principals: f.principals},
		spaceStore:              spaceStoreStub{ancestorIDs: f.ancestorIDs},
		userGroupStore:          &fakeUserGroupStore{groups: f.groups},
	}
}

func (f *suggestBatchFixture) addUser(ids ...int64) *suggestBatchFixture {
	for _, id := range ids {
		f.principals[id] = &types.Principal{ID: id, UID: fmt.Sprintf("u-%d", id), Type: enum.PrincipalTypeUser}
	}
	return f
}

// addServiceAccount registers a principal of the service account type, so a row it created is
// recognised as service owned.
func (f *suggestBatchFixture) addServiceAccount(ids ...int64) *suggestBatchFixture {
	for _, id := range ids {
		f.principals[id] = &types.Principal{
			ID:   id,
			UID:  fmt.Sprintf("svc-%d", id),
			Type: enum.PrincipalTypeServiceAccount,
		}
	}
	return f
}

// expectUpsertMany records the suggestions handed to UpsertMany so a test can assert on them.
func (f *suggestBatchFixture) expectUpsertMany() *[]*types.PullReqReviewerSuggestion {
	var captured []*types.PullReqReviewerSuggestion
	f.suggestionStore.
		On("UpsertMany", mock.Anything).
		Run(func(args mock.Arguments) {
			captured, _ = args.Get(0).([]*types.PullReqReviewerSuggestion)
		}).
		Return(nil).
		Once()
	return &captured
}

// expectExistingRow makes Find report an existing suggestion row for the principal.
func (f *suggestBatchFixture) expectExistingRow(existing *types.PullReqReviewerSuggestion) {
	f.suggestionStore.On("Find", f.prID, existing.PrincipalID).Return(existing, nil).Once()
}

// expectNoExistingRow makes Find report that the principal has no suggestion row yet.
func (f *suggestBatchFixture) expectNoExistingRow(principalID int64) {
	f.suggestionStore.
		On("Find", f.prID, principalID).
		Return((*types.PullReqReviewerSuggestion)(nil), basestore.ErrResourceNotFound).
		Once()
}

func (f *suggestBatchFixture) assertExpectations(t *testing.T) {
	t.Helper()
	f.pullreqStore.AssertExpectations(t)
	f.reviewerStore.AssertExpectations(t)
	f.suggestionStore.AssertExpectations(t)
}

func suggestInput(rows ...pullreqservice.ReviewerSuggestionInput) *pullreqservice.ReviewerSuggestBatchInput {
	return &pullreqservice.ReviewerSuggestBatchInput{Suggestions: rows}
}

func TestReviewerSuggestBatch_InvalidInput(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	session := &auth.Session{}

	_, err := ctrl.ReviewerSuggestBatch(
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
		authorizer:              authztest.AllowAuthorizer{},
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
		authorizer:              authztest.AllowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	_, err := ctrl.ReviewerSuggestBatch(
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

func TestReviewerSuggestBatch_AllAlreadyReviewersSkipsCreateMany(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).
		Return(&types.PullReq{ID: 55, Number: 7, CreatedBy: 101}, nil).Once()
	reviewerStore.On("List", int64(55)).
		Return([]*types.PullReqReviewer{{PrincipalID: 200}}, nil).Once()

	ctrl := &Controller{
		authorizer:              authztest.AllowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerStore:           reviewerStore,
		reviewerSuggestionStore: suggestionStore,
	}

	out, err := ctrl.ReviewerSuggestBatch(
		context.Background(),
		&auth.Session{Principal: types.Principal{ID: 100, UID: "u-100", Type: enum.PrincipalTypeUser}},
		"1",
		7,
		&pullreqservice.ReviewerSuggestBatchInput{ReviewerIDs: []int64{200}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output for the legacy path, got: %+v", out)
	}

	pullreqStore.AssertExpectations(t)
	reviewerStore.AssertExpectations(t)
	suggestionStore.AssertNotCalled(t, "CreateMany", mock.Anything)
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
		authorizer:              authztest.AllowAuthorizer{},
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
		authorizer:              authztest.AllowAuthorizer{},
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

func TestReviewerSuggestBatch_LegacyPathUnchanged(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)

	var created []*types.PullReqReviewerSuggestion
	f.suggestionStore.
		On("CreateMany", mock.Anything).
		Run(func(args mock.Arguments) {
			created, _ = args.Get(0).([]*types.PullReqReviewerSuggestion)
		}).
		Return(nil).
		Once()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		&pullreqservice.ReviewerSuggestBatchInput{ReviewerIDs: []int64{200}},
	)
	require.NoError(t, err)
	assert.Nil(t, out, "legacy reviewer_ids path must return a nil output")

	require.Len(t, created, 1)
	assert.Equal(t, int64(55), created[0].PullReqID)
	assert.Equal(t, int64(100), created[0].CreatedBy)
	assert.Equal(t, int64(200), created[0].PrincipalID)
	assert.Empty(t, created[0].Reason)
	assert.Empty(t, string(created[0].Kind))

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.suggestionStore.AssertNotCalled(t, "Count", mock.Anything)
	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_ServiceAccountStoresKindAndReason(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(3), nil).Once()
	f.expectNoExistingRow(200)
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testServiceSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{
			PrincipalID: 200,
			Kind:        enum.ReviewerSuggestionKindRequired,
			Reason:      "owns pkg/git",
			Rank:        new(int64(2)),
			CommitSHA:   "abc123",
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, int64(200), out.Results[0].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusCreated, out.Results[0].Status)
	assert.Empty(t, out.Results[0].Error)

	require.Len(t, *upserted, 1)
	row := (*upserted)[0]
	assert.Equal(t, int64(55), row.PullReqID)
	assert.Equal(t, int64(300), row.CreatedBy)
	assert.Equal(t, int64(200), row.PrincipalID)
	assert.Equal(t, enum.ReviewerSuggestionKindRequired, row.Kind)
	assert.Equal(t, "owns pkg/git", row.Reason)
	require.NotNil(t, row.Rank)
	assert.Equal(t, int64(2), *row.Rank)
	assert.Equal(t, "abc123", row.CommitSHA)

	f.suggestionStore.AssertNotCalled(t, "CreateMany", mock.Anything)
	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_UserCallerKindAndReasonIgnored(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(0), nil).Once()
	f.expectNoExistingRow(200)
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{
			PrincipalID: 200,
			Kind:        enum.ReviewerSuggestionKindRequired,
			Reason:      "pick me",
			Rank:        new(int64(1)),
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusCreated, out.Results[0].Status)

	require.Len(t, *upserted, 1)
	row := (*upserted)[0]
	assert.Equal(t, enum.ReviewerSuggestionKindOptional, row.Kind, "a user caller must not forge a kind")
	assert.Empty(t, row.Reason, "a user caller must not forge a reason")
	require.NotNil(t, row.Rank)
	assert.Equal(t, int64(1), *row.Rank)

	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_UserCannotOverwriteServiceCreatedRow(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200).addServiceAccount(300)
	f.suggestionStore.On("Count", int64(55)).Return(int64(1), nil).Once()
	f.expectExistingRow(&types.PullReqReviewerSuggestion{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   300,
		Kind:        enum.ReviewerSuggestionKindRequired,
		Reason:      "codeowner",
	})

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "service account")

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.assertExpectations(t)
}

// A plain row created by a service account is protected by its creator, not by its content.
func TestReviewerSuggestBatch_UserCannotOverwriteServiceCreatedRowWithoutDetail(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200).addServiceAccount(300)
	f.suggestionStore.On("Count", int64(55)).Return(int64(1), nil).Once()
	f.expectExistingRow(&types.PullReqReviewerSuggestion{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   300,
		Kind:        enum.ReviewerSuggestionKindOptional,
	})

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "service account")

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.assertExpectations(t)
}

// A user may refresh a rank bearing row they created themselves: the content is not what makes
// a row off limits, its creator is.
func TestReviewerSuggestBatch_UserCanRefreshOwnRankedRow(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(100, 200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(1), nil).Once()
	f.expectExistingRow(&types.PullReqReviewerSuggestion{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   100,
		Kind:        enum.ReviewerSuggestionKindOptional,
		Rank:        new(int64(1)),
		CommitSHA:   "abc",
	})
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200, Rank: new(int64(2))}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusUpdated, out.Results[0].Status)
	assert.Empty(t, out.Results[0].Error)

	require.Len(t, *upserted, 1)
	require.NotNil(t, (*upserted)[0].Rank)
	assert.Equal(t, int64(2), *(*upserted)[0].Rank)

	f.assertExpectations(t)
}

// created_by survives upserts, so a row a user created and a service later enriched still looks
// user owned. A user refresh must keep the stored kind and reason instead of resetting them.
func TestReviewerSuggestBatch_UserRefreshKeepsServiceEnrichedDetail(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(100, 200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(1), nil).Once()
	f.expectExistingRow(&types.PullReqReviewerSuggestion{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   100,
		Kind:        enum.ReviewerSuggestionKindRequired,
		Reason:      "codeowner",
	})
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200, Rank: new(int64(2))}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusUpdated, out.Results[0].Status)

	require.Len(t, *upserted, 1)
	row := (*upserted)[0]
	assert.Equal(t, enum.ReviewerSuggestionKindRequired, row.Kind, "a user refresh must keep the stored kind")
	assert.Equal(t, "codeowner", row.Reason, "a user refresh must keep the stored reason")

	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_UnknownPrincipalRejectedOthersSaved(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(0), nil).Once()
	f.expectNoExistingRow(200)
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(
			pullreqservice.ReviewerSuggestionInput{PrincipalID: 404},
			pullreqservice.ReviewerSuggestionInput{PrincipalID: 200},
		),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 2)

	// Sanitize orders the rows by principal id, the results follow that order.
	assert.Equal(t, int64(200), out.Results[0].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusCreated, out.Results[0].Status)
	assert.Equal(t, int64(404), out.Results[1].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[1].Status)
	assert.NotEmpty(t, out.Results[1].Error)

	require.Len(t, *upserted, 1)
	assert.Equal(t, int64(200), (*upserted)[0].PrincipalID)

	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_NoRepoAccessRejected(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.authorizer = principalDenyAuthorizer{denied: map[int64]struct{}{200: {}}}

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "permission")

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.suggestionStore.AssertNotCalled(t, "Count", mock.Anything)
	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_AuthorRejectedNotFatal(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(0), nil).Once()
	f.expectNoExistingRow(200)
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(
			pullreqservice.ReviewerSuggestionInput{PrincipalID: 101},
			pullreqservice.ReviewerSuggestionInput{PrincipalID: 200},
		),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 2)
	assert.Equal(t, int64(101), out.Results[0].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "author")
	assert.Equal(t, int64(200), out.Results[1].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusCreated, out.Results[1].Status)

	require.Len(t, *upserted, 1)
	assert.Equal(t, int64(200), (*upserted)[0].PrincipalID)

	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_AlreadyReviewerSkipped(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr, &types.PullReqReviewer{PrincipalID: 200})

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusSkipped, out.Results[0].Status)
	assert.Empty(t, out.Results[0].Error)

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.suggestionStore.AssertNotCalled(t, "Count", mock.Anything)
	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_UserGroupNotFoundRejected(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200, UserGroupID: new(int64(900))}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "could not be found")

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_UserGroupFromOtherSpaceRejected(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	// The repo's space hierarchy is 10 -> 5 -> 1, so a group living in the unrelated space 99
	// must not be accepted.
	f.groups[900] = &types.UserGroup{ID: 900, Identifier: "other-space-group", SpaceID: 99}

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200, UserGroupID: new(int64(900))}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "does not belong to the repository's space")

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.assertExpectations(t)
}

// Harness syncs user groups into the account, the org and the project space, so a group from an
// ancestor space of the repository is valid.
func TestReviewerSuggestBatch_UserGroupFromAncestorSpaceAccepted(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	// Space 1 is the account, two levels above the repo's parent space 10.
	f.groups[900] = &types.UserGroup{ID: 900, Identifier: "account-group", SpaceID: 1}
	f.suggestionStore.On("Count", int64(55)).Return(int64(0), nil).Once()
	f.expectNoExistingRow(200)
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testServiceSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200, UserGroupID: new(int64(900))}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusCreated, out.Results[0].Status)
	assert.Empty(t, out.Results[0].Error)

	require.Len(t, *upserted, 1)
	require.NotNil(t, (*upserted)[0].UserGroupID)
	assert.Equal(t, int64(900), *(*upserted)[0].UserGroupID)

	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_PerPullReqCapRejectsOverflow(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(pullreqservice.MaxSuggestionsPerPullReq), nil).Once()
	f.expectNoExistingRow(200)

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "50")

	f.suggestionStore.AssertNotCalled(t, "UpsertMany", mock.Anything)
	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_CapAllowsRefreshOfExistingRow(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200)
	f.suggestionStore.On("Count", int64(55)).Return(int64(pullreqservice.MaxSuggestionsPerPullReq), nil).Once()
	f.expectExistingRow(&types.PullReqReviewerSuggestion{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   100,
		Kind:        enum.ReviewerSuggestionKindOptional,
	})
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(pullreqservice.ReviewerSuggestionInput{PrincipalID: 200, CommitSHA: "def456"}),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 1)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusUpdated, out.Results[0].Status)

	require.Len(t, *upserted, 1)
	assert.Equal(t, int64(200), (*upserted)[0].PrincipalID)
	assert.Equal(t, "def456", (*upserted)[0].CommitSHA)

	f.assertExpectations(t)
}

func TestReviewerSuggestBatch_CapCountsNewRowsCumulatively(t *testing.T) {
	t.Parallel()

	pr := &types.PullReq{ID: 55, Number: 7, CreatedBy: 101}
	f := newSuggestBatchFixture(pr).addUser(200, 201)
	f.suggestionStore.
		On("Count", int64(55)).
		Return(int64(pullreqservice.MaxSuggestionsPerPullReq-1), nil).
		Once()
	f.expectNoExistingRow(200)
	f.expectNoExistingRow(201)
	upserted := f.expectUpsertMany()

	out, err := f.controller().ReviewerSuggestBatch(
		context.Background(),
		testSession(),
		"1",
		7,
		suggestInput(
			pullreqservice.ReviewerSuggestionInput{PrincipalID: 200},
			pullreqservice.ReviewerSuggestionInput{PrincipalID: 201},
		),
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Results, 2)

	// Only one slot is left, so the second new row must not slip through.
	assert.Equal(t, int64(200), out.Results[0].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusCreated, out.Results[0].Status)
	assert.Equal(t, int64(201), out.Results[1].PrincipalID)
	assert.Equal(t, pullreqservice.ReviewerSuggestStatusRejected, out.Results[1].Status)
	assert.Contains(t, out.Results[1].Error, "50")

	require.Len(t, *upserted, 1)
	assert.Equal(t, int64(200), (*upserted)[0].PrincipalID)

	// The cap check and the write must run in a serializable transaction, otherwise two
	// concurrent batches can both read the same count and both insert.
	assert.Contains(t, f.tx.opts, dbtx.TxSerializable)

	f.assertExpectations(t)
}

func listSuggestionsController(
	rows []*types.PullReqReviewerSuggestion,
	groups map[int64]*types.UserGroup,
) *Controller {
	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	suggestionStore := &mockstore.PullReqReviewerSuggestionStore{}
	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return(&types.PullReq{ID: 55, Number: 7}, nil).Once()
	suggestionStore.On("List", int64(55), mock.Anything).Return(rows, nil).Once()

	return &Controller{
		tx:                      &passthroughTx{},
		authorizer:              authztest.AllowAuthorizer{},
		repoFinder:              testRepoFinder(repo),
		pullreqStore:            pullreqStore,
		reviewerSuggestionStore: suggestionStore,
		userGroupStore:          &fakeUserGroupStore{groups: groups},
		principalInfoCache: &fakePrincipalInfoCache{infos: map[int64]*types.PrincipalInfo{
			200: {ID: 200, UID: "u-200"},
			300: {ID: 300, UID: "svc-300"},
		}},
	}
}

func TestListSuggestedReviewers_ReturnsDetailFields(t *testing.T) {
	t.Parallel()

	ctrl := listSuggestionsController([]*types.PullReqReviewerSuggestion{{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   300,
		Kind:        enum.ReviewerSuggestionKindRequired,
		Reason:      "codeowner",
		Rank:        new(int64(3)),
		CommitSHA:   "deadbeef",
	}}, nil)

	out, _, err := ctrl.ListSuggestedReviewers(
		context.Background(), testSession(), "1", 7, types.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, out.Suggestions, 1)
	item := out.Suggestions[0]
	assert.Equal(t, enum.ReviewerSuggestionKindRequired, item.Kind)
	assert.Equal(t, "codeowner", item.Reason)
	require.NotNil(t, item.Rank)
	assert.Equal(t, int64(3), *item.Rank)
	assert.Equal(t, "deadbeef", item.CommitSHA)
	assert.Nil(t, item.UserGroup)
}

func TestListSuggestedReviewers_ResolvesUserGroup(t *testing.T) {
	t.Parallel()

	ctrl := listSuggestionsController([]*types.PullReqReviewerSuggestion{{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   300,
		UserGroupID: new(int64(900)),
	}}, map[int64]*types.UserGroup{900: {ID: 900, Identifier: "platform"}})

	out, _, err := ctrl.ListSuggestedReviewers(
		context.Background(), testSession(), "1", 7, types.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, out.Suggestions, 1)
	require.NotNil(t, out.Suggestions[0].UserGroup)
	assert.Equal(t, "platform", out.Suggestions[0].UserGroup.Identifier)
}

func TestListSuggestedReviewers_MissingUserGroupLeavesItemWithoutGroup(t *testing.T) {
	t.Parallel()

	ctrl := listSuggestionsController([]*types.PullReqReviewerSuggestion{{
		PullReqID:   55,
		PrincipalID: 200,
		CreatedBy:   300,
		UserGroupID: new(int64(900)),
	}}, map[int64]*types.UserGroup{})

	out, _, err := ctrl.ListSuggestedReviewers(
		context.Background(), testSession(), "1", 7, types.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, out.Suggestions, 1)
	assert.Nil(t, out.Suggestions[0].UserGroup)
}

func TestListSuggestedReviewers_SharedUserGroupIsMappedOnce(t *testing.T) {
	t.Parallel()

	ctrl := listSuggestionsController([]*types.PullReqReviewerSuggestion{
		{PullReqID: 55, PrincipalID: 200, CreatedBy: 300, UserGroupID: new(int64(900))},
		{PullReqID: 55, PrincipalID: 300, CreatedBy: 300, UserGroupID: new(int64(900))},
	}, map[int64]*types.UserGroup{900: {ID: 900, Identifier: "platform"}})

	out, _, err := ctrl.ListSuggestedReviewers(
		context.Background(), testSession(), "1", 7, types.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, out.Suggestions, 2)
	for _, item := range out.Suggestions {
		require.NotNil(t, item.UserGroup)
		assert.Equal(t, "platform", item.UserGroup.Identifier)
	}

	groupStore, ok := ctrl.userGroupStore.(*fakeUserGroupStore)
	require.True(t, ok)
	assert.Equal(t, [][]int64{{900}}, groupStore.calls)
}

func TestListSuggestedReviewers_UnrankedItemHasNilRank(t *testing.T) {
	t.Parallel()

	ctrl := listSuggestionsController([]*types.PullReqReviewerSuggestion{
		{PullReqID: 55, PrincipalID: 200, CreatedBy: 300, Rank: new(int64(1))},
		{PullReqID: 55, PrincipalID: 300, CreatedBy: 300},
	}, nil)

	out, _, err := ctrl.ListSuggestedReviewers(
		context.Background(), testSession(), "1", 7, types.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, out.Suggestions, 2)
	require.NotNil(t, out.Suggestions[0].Rank)
	assert.Equal(t, int64(1), *out.Suggestions[0].Rank)
	assert.Nil(t, out.Suggestions[1].Rank)
}
