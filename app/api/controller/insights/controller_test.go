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

package insights

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/auth/authz/authztest"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/insights/insightstest"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type repoCache map[int64]*types.RepositoryCore

func (c repoCache) Stats() (int64, int64)            { return 0, 0 }
func (c repoCache) Evict(_ context.Context, _ int64) {}
func (c repoCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if r, ok := c[id]; ok {
		return r, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type urlStub struct{ url.Provider }

func (urlStub) GetAPIProto(context.Context) string    { return "https" }
func (urlStub) GetAPIHostname(context.Context) string { return "example.com" }

type auditEntry struct {
	resource  audit.Resource
	action    audit.Action
	spacePath string
	event     audit.Event
}

type recordingAudit struct{ entries []auditEntry }

func (a *recordingAudit) Log(
	_ context.Context, _ types.Principal, r audit.Resource, action audit.Action, spacePath string,
	opts ...audit.Option,
) error {
	e := audit.Event{}
	for _, o := range opts {
		o.Apply(&e)
	}
	a.entries = append(a.entries, auditEntry{resource: r, action: action, spacePath: spacePath, event: e})
	return nil
}

type testEnv struct {
	ctrl    *Controller
	store   *insightstest.Store
	audit   *recordingAudit
	session *auth.Session
}

func newTestEnv(authorizer authz.Authorizer) *testEnv {
	st := insightstest.New()
	rec := &recordingAudit{}
	finder := refcache.NewRepoFinder(nil, nil,
		repoCache{
			1: {ID: 1, Identifier: "r1", Path: "space/r1"},
			2: {ID: 2, Identifier: "r2", Path: "space/r2"},
		}, nil, storecache.Evictor[*types.RepositoryCore]{})
	svc := insights.NewService(insights.Config{}, st.ProjectStore(), st.IssueStore(), st.EventStore())
	return &testEnv{
		ctrl:    NewController(authorizer, finder, urlStub{}, svc, rec),
		store:   st,
		audit:   rec,
		session: &auth.Session{Principal: types.Principal{ID: 7, UID: "alice"}},
	}
}

func ptr[T any](v T) *T { return &v }

func TestProjectLifecycleAndAudit(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.AllowAuthorizer{})

	created, err := e.ctrl.CreateProject(ctx, e.session, "1", &types.InsightsProjectCreateInput{Identifier: "api"})
	if err != nil {
		t.Fatal(err)
	}

	upd, err := e.ctrl.UpdateProject(ctx, e.session, "1", created.ID, &types.InsightsProjectUpdateInput{
		Enabled: ptr(false), RateLimitPerMinute: ptr(10), SampleRate: ptr(0.5),
	})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Enabled || upd.RateLimitPerMinute != 10 || upd.SampleRate != 0.5 {
		t.Fatalf("unexpected update result %+v", upd)
	}

	rotated, err := e.ctrl.RotateProjectKey(ctx, e.session, "1", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.PublicKey == created.PublicKey || !strings.Contains(rotated.DSN, rotated.PublicKey) {
		t.Fatalf("key not rotated into the DSN: %+v", rotated)
	}

	if err = e.ctrl.DeleteProject(ctx, e.session, "1", created.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.store.Projects[created.ID]; ok {
		t.Fatal("project still exists")
	}

	wantActions := []audit.Action{audit.ActionCreated, audit.ActionUpdated, audit.ActionUpdated, audit.ActionDeleted}
	if len(e.audit.entries) != len(wantActions) {
		t.Fatalf("got %d audit entries, want %d", len(e.audit.entries), len(wantActions))
	}
	for i, en := range e.audit.entries {
		if en.action != wantActions[i] || en.resource.Type != audit.ResourceTypeInsightsProject ||
			en.resource.Identifier != "api" || en.spacePath != "space" {
			t.Errorf("entry %d unexpected: %+v", i, en)
		}
	}
	if e.audit.entries[2].event.Data["operation"] != "rotate_key" {
		t.Error("rotate entry not marked")
	}
}

func TestProjectOfOtherRepoIsNotFound(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.AllowAuthorizer{})
	p, err := e.ctrl.CreateProject(ctx, e.session, "1", &types.InsightsProjectCreateInput{Identifier: "api"})
	if err != nil {
		t.Fatal(err)
	}
	n := len(e.audit.entries)

	if _, err = e.ctrl.RotateProjectKey(ctx, e.session, "2", p.ID); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Fatalf("rotate: got %v, want not found", err)
	}
	if err = e.ctrl.DeleteProject(ctx, e.session, "2", p.ID); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Fatalf("delete: got %v, want not found", err)
	}
	if _, err = e.ctrl.UpdateProject(ctx, e.session, "2", p.ID, &types.InsightsProjectUpdateInput{}); !errors.Is(
		err, gitness_store.ErrResourceNotFound) {
		t.Fatalf("update: got %v, want not found", err)
	}
	if len(e.audit.entries) != n {
		t.Error("failed operations must not be audited")
	}
}

func TestUpdateProjectValidation(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.AllowAuthorizer{})
	p, _ := e.ctrl.CreateProject(ctx, e.session, "1", &types.InsightsProjectCreateInput{Identifier: "api"})

	for _, in := range []*types.InsightsProjectUpdateInput{
		{SampleRate: ptr(1.5)},
		{SampleRate: ptr(-0.1)},
		{RateLimitPerMinute: ptr(-1)},
		{RateLimitPerMinute: ptr(insights.MaxRateLimitPerMinute + 1)},
	} {
		if _, err := e.ctrl.UpdateProject(ctx, e.session, "1", p.ID, in); err == nil {
			t.Errorf("expected validation error for %+v", in)
		}
	}
}

func seedIssues(e *testEnv) {
	e.store.Projects[1] = &types.InsightsProject{ID: 1, RepoID: 1, Identifier: "api"}
	e.store.Projects[2] = &types.InsightsProject{ID: 2, RepoID: 1, Identifier: "web"}
	e.store.Projects[3] = &types.InsightsProject{ID: 3, RepoID: 2, Identifier: "other"}
	add := func(id, project, repo, lastSeen, times int64, status enum.InsightsIssueStatus, level, title, rel string) {
		e.store.Issues[id] = &types.InsightsIssue{
			ID: id, ProjectID: project, RepoID: repo, LastSeen: lastSeen, TimesSeen: times,
			Status: status, Level: level, Title: title, FirstRelease: rel, LastRelease: rel,
		}
	}
	add(1, 1, 1, 100, 5, enum.InsightsIssueStatusUnresolved, "error", "TypeError: boom", "1.0.0")
	add(2, 1, 1, 300, 1, enum.InsightsIssueStatusResolved, "warning", "ValueError: bad", "1.1.0")
	add(3, 2, 1, 200, 9, enum.InsightsIssueStatusIgnored, "error", "typeerror: other", "1.1.0")
	add(4, 3, 2, 400, 1, enum.InsightsIssueStatusUnresolved, "error", "TypeError: foreign", "1.0.0")
}

func issueIDs(list []*types.InsightsIssue) []int64 {
	out := make([]int64, len(list))
	for i, x := range list {
		out[i] = x.ID
	}
	return out
}

func equalIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestListIssuesFilters(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.AllowAuthorizer{})
	seedIssues(e)

	page := types.Pagination{Page: 1, Size: 10}
	list := types.ListQueryFilter{Pagination: page}
	tests := []struct {
		name   string
		filter types.InsightsIssueFilter
		want   []int64
	}{
		{"default sort is last_seen desc", types.InsightsIssueFilter{ListQueryFilter: list}, []int64{2, 3, 1}},
		{"asc", types.InsightsIssueFilter{ListQueryFilter: list, Order: enum.OrderAsc}, []int64{1, 3, 2}},
		{"times_seen", types.InsightsIssueFilter{ListQueryFilter: list, Sort: enum.InsightsIssueSortTimesSeen},
			[]int64{3, 1, 2}},
		{"project", types.InsightsIssueFilter{ListQueryFilter: list, ProjectID: 2}, []int64{3}},
		{"status", types.InsightsIssueFilter{ListQueryFilter: list,
			Statuses: []enum.InsightsIssueStatus{enum.InsightsIssueStatusResolved, enum.InsightsIssueStatusIgnored}},
			[]int64{2, 3}},
		{"level", types.InsightsIssueFilter{ListQueryFilter: list, Level: "warning"}, []int64{2}},
		{"release", types.InsightsIssueFilter{ListQueryFilter: list, Release: "1.1.0"}, []int64{2, 3}},
		{"title query", types.InsightsIssueFilter{ListQueryFilter: types.ListQueryFilter{
			Query: "TYPEERROR", Pagination: page}}, []int64{3, 1}},
		{"page 2", types.InsightsIssueFilter{ListQueryFilter: types.ListQueryFilter{
			Pagination: types.Pagination{Page: 2, Size: 2}}}, []int64{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := e.ctrl.ListIssues(ctx, e.session, "1", &tt.filter)
			if err != nil {
				t.Fatal(err)
			}
			if !equalIDs(issueIDs(got), tt.want) {
				t.Errorf("got %v, want %v", issueIDs(got), tt.want)
			}
		})
	}

	_, total, err := e.ctrl.ListIssues(ctx, e.session, "1", &types.InsightsIssueFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 1}}})
	if err != nil || total != 3 {
		t.Fatalf("total = %d, err = %v; want 3", total, err)
	}
}

func TestIssueEvents(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.AllowAuthorizer{})
	seedIssues(e)
	for i := int64(1); i <= 5; i++ {
		e.store.Events[i] = &types.InsightsEvent{ID: i, IssueID: 1, ProjectID: 1, Timestamp: i * 10,
			Payload: `{"event_id":"x"}`}
	}

	evs, total, err := e.ctrl.ListIssueEvents(ctx, e.session, "1", 1, types.Pagination{Page: 1, Size: 2})
	if err != nil || total != 5 || len(evs) != 2 || evs[0].Timestamp != 50 || evs[1].Timestamp != 40 {
		t.Fatalf("got %v total %d err %v", evs, total, err)
	}
	if evs[0].Payload == "" {
		t.Error("payload must be returned")
	}
	evs, _, _ = e.ctrl.ListIssueEvents(ctx, e.session, "1", 1, types.Pagination{Page: 3, Size: 2})
	if len(evs) != 1 || evs[0].Timestamp != 10 {
		t.Errorf("unexpected last page %v", evs)
	}

	if _, _, err = e.ctrl.ListIssueEvents(ctx, e.session, "1", 4, types.Pagination{Page: 1, Size: 2}); !errors.Is(
		err, gitness_store.ErrResourceNotFound) {
		t.Errorf("issue of another repo: got %v, want not found", err)
	}
	if _, err = e.ctrl.FindIssue(ctx, e.session, "1", 4); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Errorf("find issue of another repo: got %v, want not found", err)
	}
}

func TestUpdateIssue(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.AllowAuthorizer{})
	seedIssues(e)

	got, err := e.ctrl.UpdateIssue(ctx, e.session, "1", 1, &types.InsightsIssueUpdateInput{
		Status: enum.InsightsIssueStatusResolved, ResolvedInRelease: ptr("1.2.0")})
	if err != nil || got.Status != enum.InsightsIssueStatusResolved || got.ResolvedInRelease != "1.2.0" {
		t.Fatalf("resolve: %+v, %v", got, err)
	}

	got, err = e.ctrl.UpdateIssue(ctx, e.session, "1", 1, &types.InsightsIssueUpdateInput{
		Status: enum.InsightsIssueStatusIgnored})
	if err != nil || got.Status != enum.InsightsIssueStatusIgnored || got.ResolvedInRelease != "" {
		t.Fatalf("ignore must clear the release: %+v, %v", got, err)
	}

	bad := []*types.InsightsIssueUpdateInput{
		{Status: enum.InsightsIssueStatusRegressed},
		{Status: "bogus"},
		{},
		{Status: enum.InsightsIssueStatusIgnored, ResolvedInRelease: ptr("1.0.0")},
	}
	for _, in := range bad {
		if _, err = e.ctrl.UpdateIssue(ctx, e.session, "1", 1, in); err == nil {
			t.Errorf("expected error for %+v", in)
		}
	}

	if _, err = e.ctrl.UpdateIssue(ctx, e.session, "1", 4, &types.InsightsIssueUpdateInput{
		Status: enum.InsightsIssueStatusResolved}); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Errorf("foreign issue: got %v, want not found", err)
	}
}

func TestPermissionsDenied(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(authztest.DenyAuthorizer{})
	seedIssues(e)

	if _, _, err := e.ctrl.ListIssues(ctx, e.session, "1", &types.InsightsIssueFilter{}); err == nil {
		t.Error("list issues must be denied")
	}
	if _, err := e.ctrl.FindIssue(ctx, e.session, "1", 1); err == nil {
		t.Error("find issue must be denied")
	}
	if _, _, err := e.ctrl.ListIssueEvents(ctx, e.session, "1", 1, types.Pagination{}); err == nil {
		t.Error("list events must be denied")
	}
	if _, err := e.ctrl.UpdateIssue(ctx, e.session, "1", 1, &types.InsightsIssueUpdateInput{
		Status: enum.InsightsIssueStatusResolved}); err == nil {
		t.Error("update issue must be denied")
	}
	if _, err := e.ctrl.UpdateProject(ctx, e.session, "1", 1, &types.InsightsProjectUpdateInput{}); err == nil {
		t.Error("update project must be denied")
	}
	if _, err := e.ctrl.RotateProjectKey(ctx, e.session, "1", 1); err == nil {
		t.Error("rotate must be denied")
	}
	if err := e.ctrl.DeleteProject(ctx, e.session, "1", 1); err == nil {
		t.Error("delete must be denied")
	}
	if len(e.audit.entries) != 0 || e.store.Projects[1] == nil {
		t.Error("denied operations must not change anything")
	}
}
