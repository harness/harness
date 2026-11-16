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

// Regression coverage for CODE-6689: "Reviewers not returned in direct list but appear in
// search results".
//
// Root cause: the combined reviewer list endpoint enriches every user group reviewer by
// resolving the group against harness-core (userGroupService.MapGroupIDsToPrincipals). Before
// the fix any single failure there - e.g. a group deleted in harness-core after it was added
// as a reviewer, or a transient harness-core error - made the whole endpoint error out, so the
// UI showed no reviewers at all (neither individual nor user group reviewers). The search path
// is self-healing and kept listing groups, hence the asymmetry reported in the ticket.
//
// These tests exercise the enrichment path (Controller.listUserGroupReviewers) with lightweight
// fakes and assert that reviewers are still returned when group resolution fails.
package pullreq

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	appstore "github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const testSourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// --- fakes (only the methods hit by listUserGroupReviewers are implemented) ---

type fakeUserGroupReviewerStore struct {
	appstore.UserGroupReviewerStore
	reviewers []*types.UserGroupReviewer
	err       error
}

func (f *fakeUserGroupReviewerStore) List(context.Context, int64) ([]*types.UserGroupReviewer, error) {
	return f.reviewers, f.err
}

type fakeUserGroupStore struct {
	appstore.UserGroupStore
	groups map[int64]*types.UserGroup
	err    error
}

func (f *fakeUserGroupStore) Map(context.Context, []int64) (map[int64]*types.UserGroup, error) {
	return f.groups, f.err
}

type fakePrincipalInfoCache struct {
	appstore.PrincipalInfoCache
	infos map[int64]*types.PrincipalInfo
}

func (f *fakePrincipalInfoCache) Map(_ context.Context, _ []int64) (map[int64]*types.PrincipalInfo, error) {
	return f.infos, nil
}

type fakeUserGroupService struct {
	principals map[int64][]*types.Principal
	err        error
}

func (f *fakeUserGroupService) List(
	context.Context, *types.ListQueryFilter, *types.SpaceCore,
) ([]*types.UserGroupInfo, error) {
	return nil, nil
}

func (f *fakeUserGroupService) ListUserIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}

func (f *fakeUserGroupService) MapGroupIDsToPrincipals(
	context.Context, []int64,
) (map[int64][]*types.Principal, error) {
	return f.principals, f.err
}

// newTestController wires only the dependencies that listUserGroupReviewers uses.
func newTestController(
	ugReviewers *fakeUserGroupReviewerStore,
	ugStore *fakeUserGroupStore,
	piCache *fakePrincipalInfoCache,
	ugSvc *fakeUserGroupService,
) *Controller {
	return &Controller{
		userGroupReviewerStore: ugReviewers,
		userGroupStore:         ugStore,
		principalInfoCache:     piCache,
		userGroupService:       ugSvc,
	}
}

func TestListUserGroupReviewers_GroupResolutionFailureStillReturnsReviewers(t *testing.T) {
	// CODE-6689: a stale/unresolvable group must not hide the reviewer list.
	pr := &types.PullReq{ID: 1, SourceSHA: testSourceSHA}

	// One individual reviewer who approved on the source SHA.
	reviewers := []*types.PullReqReviewer{
		{
			PrincipalID:    100,
			ReviewDecision: enum.PullReqReviewDecisionApproved,
			SHA:            testSourceSHA,
		},
	}

	ctrl := newTestController(
		&fakeUserGroupReviewerStore{
			reviewers: []*types.UserGroupReviewer{
				{PullReqID: 1, UserGroupID: 10, CreatedBy: 100},
			},
		},
		&fakeUserGroupStore{
			groups: map[int64]*types.UserGroup{
				10: {ID: 10, Identifier: "my-group", Name: "My Group"},
			},
		},
		&fakePrincipalInfoCache{
			infos: map[int64]*types.PrincipalInfo{
				100: {ID: 100, UID: "adder"},
			},
		},
		// Simulate harness-core resolution failure for the group.
		&fakeUserGroupService{err: context.DeadlineExceeded},
	)

	got, err := ctrl.listUserGroupReviewers(context.Background(), pr, reviewers)
	if err != nil {
		t.Fatalf("expected no error when group resolution fails, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the user group reviewer to still be returned, got %d", len(got))
	}
	if got[0].UserGroupID != 10 {
		t.Errorf("unexpected user group id: %d", got[0].UserGroupID)
	}
	// Metadata available in the DB should still be populated.
	if got[0].UserGroup.Identifier != "my-group" {
		t.Errorf("expected group identifier to be populated, got %q", got[0].UserGroup.Identifier)
	}
	if got[0].AddedBy.UID != "adder" {
		t.Errorf("expected addedBy to be populated, got %q", got[0].AddedBy.UID)
	}
	// Without resolvable members the decision degrades to pending (not an error).
	if got[0].Decision != enum.PullReqReviewDecisionPending {
		t.Errorf("expected pending decision, got %q", got[0].Decision)
	}
}

func TestListUserGroupReviewers_MissingUserGroupRowStillReturnsReviewer(t *testing.T) {
	// Defensive: even if the usergroups row cannot be mapped, the reviewer must not be dropped
	// (and the old code would nil-panic dereferencing the absent map entry).
	pr := &types.PullReq{ID: 1, SourceSHA: testSourceSHA}

	ctrl := newTestController(
		&fakeUserGroupReviewerStore{
			reviewers: []*types.UserGroupReviewer{
				{PullReqID: 1, UserGroupID: 10, CreatedBy: 100},
			},
		},
		&fakeUserGroupStore{err: gitnessstore.ErrResourceNotFound},
		&fakePrincipalInfoCache{infos: map[int64]*types.PrincipalInfo{}},
		&fakeUserGroupService{err: gitnessstore.ErrResourceNotFound},
	)

	got, err := ctrl.listUserGroupReviewers(context.Background(), pr, nil)
	if err != nil {
		t.Fatalf("expected no error when usergroup row is missing, got: %v", err)
	}
	if len(got) != 1 || got[0].UserGroupID != 10 {
		t.Fatalf("expected the user group reviewer to still be returned, got %#v", got)
	}
	// Even without metadata the response must carry a usable identity so the UI can key and
	// remove each stale group (user_group.id must not default to zero).
	if got[0].UserGroup.ID != 10 {
		t.Errorf("expected user group id fallback to 10, got %d", got[0].UserGroup.ID)
	}
}

func TestListUserGroupReviewers_HappyPathComputesDecision(t *testing.T) {
	pr := &types.PullReq{ID: 1, SourceSHA: testSourceSHA}

	reviewers := []*types.PullReqReviewer{
		{PrincipalID: 100, ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: testSourceSHA},
	}

	ctrl := newTestController(
		&fakeUserGroupReviewerStore{
			reviewers: []*types.UserGroupReviewer{
				{PullReqID: 1, UserGroupID: 10, CreatedBy: 100},
			},
		},
		&fakeUserGroupStore{
			groups: map[int64]*types.UserGroup{
				10: {ID: 10, Identifier: "my-group", Name: "My Group"},
			},
		},
		&fakePrincipalInfoCache{infos: map[int64]*types.PrincipalInfo{100: {ID: 100, UID: "adder"}}},
		&fakeUserGroupService{
			principals: map[int64][]*types.Principal{
				10: {{ID: 100}},
			},
		},
	)

	got, err := ctrl.listUserGroupReviewers(context.Background(), pr, reviewers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one user group reviewer, got %d", len(got))
	}
	if got[0].Decision != enum.PullReqReviewDecisionApproved {
		t.Errorf("expected approved compound decision, got %q", got[0].Decision)
	}
	if got[0].SHA != testSourceSHA {
		t.Errorf("expected decision SHA to be the source SHA, got %q", got[0].SHA)
	}
}

// --- fakes for the full ReviewersListCombined path ---

// fakeAuthorizer grants every check so the test focuses on the reviewer logic, not access control.
type fakeAuthorizer struct{}

func (fakeAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return true, nil
}

func (fakeAuthorizer) CheckAll(
	context.Context, *auth.Session, ...types.PermissionCheck,
) (bool, error) {
	return true, nil
}

func (fakeAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	out := make([]bool, len(checks))
	for i := range out {
		out[i] = true
	}
	return out, nil
}

var _ authz.Authorizer = fakeAuthorizer{}

// fakeRepoIDCache serves a single repo by ID so RepoFinder.FindByRef resolves a numeric ref.
type fakeRepoIDCache struct {
	repo *types.RepositoryCore
}

func (fakeRepoIDCache) Stats() (int64, int64) { return 0, 0 }
func (f fakeRepoIDCache) Get(context.Context, int64) (*types.RepositoryCore, error) {
	return f.repo, nil
}
func (fakeRepoIDCache) Evict(context.Context, int64) {}

type fakePullReqStore struct {
	appstore.PullReqStore
	pr *types.PullReq
}

func (f *fakePullReqStore) FindByNumber(context.Context, int64, int64) (*types.PullReq, error) {
	return f.pr, nil
}

type fakeReviewerStore struct {
	appstore.PullReqReviewerStore
	reviewers []*types.PullReqReviewer
}

func (f *fakeReviewerStore) List(context.Context, int64) ([]*types.PullReqReviewer, error) {
	return f.reviewers, nil
}

// TestReviewersListCombined_GroupResolutionFailureStillReturnsReviewers drives the full public
// endpoint (auth -> repo -> pr -> reviewers -> user groups) and is the closest in-process
// reproduction of CODE-6689: a failing group resolution must not blank out the direct reviewer
// list the UI reads from this endpoint.
func TestReviewersListCombined_GroupResolutionFailureStillReturnsReviewers(t *testing.T) {
	repo := &types.RepositoryCore{ID: 1, Path: "my-space/my-repo", State: enum.RepoStateActive}
	pr := &types.PullReq{ID: 7, Number: 42, SourceSHA: testSourceSHA}

	reviewers := []*types.PullReqReviewer{
		{PrincipalID: 100, ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: testSourceSHA},
	}

	ctrl := &Controller{
		authorizer: fakeAuthorizer{},
		repoFinder: refcache.NewRepoFinder(
			nil, nil, fakeRepoIDCache{repo: repo}, nil,
			storecache.Evictor[*types.RepositoryCore]{},
		),
		pullreqStore:  &fakePullReqStore{pr: pr},
		reviewerStore: &fakeReviewerStore{reviewers: reviewers},
		userGroupReviewerStore: &fakeUserGroupReviewerStore{
			reviewers: []*types.UserGroupReviewer{
				{PullReqID: 7, UserGroupID: 10, CreatedBy: 100},
			},
		},
		userGroupStore: &fakeUserGroupStore{
			groups: map[int64]*types.UserGroup{
				10: {ID: 10, Identifier: "my-group", Name: "My Group"},
			},
		},
		principalInfoCache: &fakePrincipalInfoCache{
			infos: map[int64]*types.PrincipalInfo{100: {ID: 100, UID: "adder"}},
		},
		// harness-core resolution fails: before the fix this 500'd the whole endpoint.
		userGroupService: &fakeUserGroupService{err: context.DeadlineExceeded},
	}

	// Numeric repoRef so RepoFinder resolves straight through fakeRepoIDCache.
	resp, err := ctrl.ReviewersListCombined(context.Background(), &auth.Session{}, "1", 42)
	if err != nil {
		t.Fatalf("expected no error when group resolution fails, got: %v", err)
	}
	// The reported symptom: the direct reviewer list went empty. It must still be present.
	if len(resp.Reviewers) != 1 || resp.Reviewers[0].PrincipalID != 100 {
		t.Fatalf("expected the individual reviewer to still be returned, got %#v", resp.Reviewers)
	}
	// And the user group reviewer still surfaces, just with a degraded (pending) decision.
	if len(resp.UserGroupReviewers) != 1 {
		t.Fatalf("expected the user group reviewer to still be returned, got %d", len(resp.UserGroupReviewers))
	}
	if resp.UserGroupReviewers[0].Decision != enum.PullReqReviewDecisionPending {
		t.Errorf("expected pending decision, got %q", resp.UserGroupReviewers[0].Decision)
	}
}
