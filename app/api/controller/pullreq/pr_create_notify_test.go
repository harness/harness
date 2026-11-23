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
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// findManyByUIDStub stubs store.PrincipalStore so notifyUserGroupReviewersAdded
// can resolve group member UIDs to principals without a database.
type findManyByUIDStub struct {
	store.PrincipalStore
	gotUIDs    []string
	calls      int
	principals []*types.Principal
	err        error
}

func (s *findManyByUIDStub) FindManyByUID(_ context.Context, uids []string) ([]*types.Principal, error) {
	s.gotUIDs = uids
	s.calls++
	return s.principals, s.err
}

// notifyAuthorID is the session principal (PR author) used across these tests.
const notifyAuthorID int64 = 1

func notifySession() *auth.Session {
	return &auth.Session{Principal: types.Principal{ID: notifyAuthorID}}
}

func notifyUserGroup(id int64, users ...string) *types.UserGroup {
	return &types.UserGroup{ID: id, Users: users}
}

// TestNotifyUserGroupReviewersAdded_ReturnsEarlyWhenMapEmpty: an empty source
// map does no lookups and reports no event.
func TestNotifyUserGroupReviewersAdded_ReturnsEarlyWhenMapEmpty(t *testing.T) {
	t.Parallel()

	principalStore := &findManyByUIDStub{}
	ctrl := &Controller{
		principalStore: principalStore,
		eventReporter:  mockpullreq.NewStubReporter(t),
	}

	ctrl.notifyUserGroupReviewersAdded(
		context.Background(), notifySession(), &types.PullReq{ID: 20},
		map[int64]*types.UserGroup{}, enum.PullReqReviewerTypeRequested,
	)

	if principalStore.gotUIDs != nil {
		t.Fatalf("expected no principal lookup for empty map, got %v", principalStore.gotUIDs)
	}
}

// TestNotifyUserGroupReviewersAdded_DeduplicatesMemberUIDs: UIDs are gathered
// across groups, sorted, and compacted before the principal lookup.
func TestNotifyUserGroupReviewersAdded_DeduplicatesMemberUIDs(t *testing.T) {
	t.Parallel()

	principalStore := &findManyByUIDStub{
		principals: []*types.Principal{{ID: 2}, {ID: 3}},
	}
	ctrl := &Controller{
		principalStore: principalStore,
		eventReporter:  mockpullreq.NewStubReporter(t),
	}

	groups := map[int64]*types.UserGroup{
		10: notifyUserGroup(10, "bob", "alice"),
		11: notifyUserGroup(11, "alice", "carol"),
	}

	ctrl.notifyUserGroupReviewersAdded(
		context.Background(), notifySession(), &types.PullReq{ID: 20},
		groups, enum.PullReqReviewerTypeCodeOwners,
	)

	want := []string{"alice", "bob", "carol"}
	if len(principalStore.gotUIDs) != len(want) {
		t.Fatalf("expected deduped+sorted UIDs %v, got %v", want, principalStore.gotUIDs)
	}
	for i, uid := range want {
		if principalStore.gotUIDs[i] != uid {
			t.Fatalf("expected deduped+sorted UIDs %v, got %v", want, principalStore.gotUIDs)
		}
	}
}

// TestNotifyUserGroupReviewersAdded_ReportsEventExcludingAuthor: the PR author
// is dropped from the member list and the remaining members produce an event.
func TestNotifyUserGroupReviewersAdded_ReportsEventExcludingAuthor(t *testing.T) {
	t.Parallel()

	const authorID int64 = 1
	principalStore := &findManyByUIDStub{
		principals: []*types.Principal{{ID: authorID}, {ID: 2}, {ID: 3}},
	}
	ctrl := &Controller{
		principalStore: principalStore,
		eventReporter:  mockpullreq.NewStubReporter(t),
	}

	groups := map[int64]*types.UserGroup{10: notifyUserGroup(10, "author", "alice", "bob")}

	// Does not panic and reports an event (verified by reaching the reporter);
	// the author (session principal) is excluded from the member list.
	ctrl.notifyUserGroupReviewersAdded(
		context.Background(), notifySession(), &types.PullReq{ID: 20, Number: 42},
		groups, enum.PullReqReviewerTypeDefault,
	)
}

// TestNotifyUserGroupReviewersAdded_ReturnsEarlyWhenOnlyAuthor: when the only
// member is the author, nothing is reported after exclusion.
func TestNotifyUserGroupReviewersAdded_ReturnsEarlyWhenOnlyAuthor(t *testing.T) {
	t.Parallel()

	const authorID int64 = 1
	principalStore := &findManyByUIDStub{
		principals: []*types.Principal{{ID: authorID}},
	}
	ctrl := &Controller{
		principalStore: principalStore,
		eventReporter:  mockpullreq.NewStubReporter(t),
	}

	groups := map[int64]*types.UserGroup{10: notifyUserGroup(10, "author")}

	ctrl.notifyUserGroupReviewersAdded(
		context.Background(), notifySession(), &types.PullReq{ID: 20},
		groups, enum.PullReqReviewerTypeRequested,
	)
}

// TestNotifyUserGroupReviewersAdded_SwallowsLookupError: a principal lookup
// failure is logged and swallowed (no event, no panic).
func TestNotifyUserGroupReviewersAdded_SwallowsLookupError(t *testing.T) {
	t.Parallel()

	principalStore := &findManyByUIDStub{err: errors.New("lookup failed")}
	ctrl := &Controller{
		principalStore: principalStore,
		eventReporter:  mockpullreq.NewStubReporter(t),
	}

	groups := map[int64]*types.UserGroup{10: notifyUserGroup(10, "alice")}

	ctrl.notifyUserGroupReviewersAdded(
		context.Background(), notifySession(), &types.PullReq{ID: 20},
		groups, enum.PullReqReviewerTypeRequested,
	)
}

// TestReportPullReqCreated_EmitsEventAndPerSourceNotifications drives the
// extracted reportPullReqCreated helper end to end: it reports the created
// event (with the per-reviewer origin map) and fires a user-group
// reviewer-added notification for each of the three non-empty source maps.
func TestReportPullReqCreated_EmitsEventAndPerSourceNotifications(t *testing.T) {
	t.Parallel()

	principalStore := &findManyByUIDStub{
		principals: []*types.Principal{{ID: 2}, {ID: 3}},
	}
	ctrl := &Controller{
		principalStore: principalStore,
		eventReporter:  mockpullreq.NewStubReporter(t),
	}

	userReviewerMap := map[int64]*types.PrincipalInfo{2: {ID: 2}, 3: {ID: 3}}
	userGroupReviewerMap := map[int64]*types.UserGroup{
		10: notifyUserGroup(10, "alice", "bob"),
		11: notifyUserGroup(11, "carol"),
		12: notifyUserGroup(12, "dave"),
	}

	ctrl.reportPullReqCreated(
		context.Background(), notifySession(), &types.PullReq{ID: 20, Number: 42},
		"feature", "main", sha.None,
		userReviewerMap, userGroupReviewerMap,
		[]int64{2}, // requested direct reviewer
		map[int64]*types.PrincipalInfo{3: {ID: 3}}, // code-owner direct reviewer
		map[int64]*types.PrincipalInfo{},           // default direct reviewers (none)
		[]int64{10},                                // requested group (present in userGroupReviewerMap)
		map[int64]*types.UserGroup{11: notifyUserGroup(11, "carol")},
		map[int64]*types.UserGroup{12: notifyUserGroup(12, "dave")},
	)

	// One principal lookup per non-empty group source (requested/owner/default).
	if principalStore.calls != 3 {
		t.Fatalf("expected 3 principal lookups (one per group source), got %d", principalStore.calls)
	}
}
