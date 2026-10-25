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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ctrlinsights "github.com/harness/gitness/app/api/controller/insights"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
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

	"github.com/go-chi/chi/v5"
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

func newRouter(t *testing.T, st *insightstest.Store) http.Handler {
	t.Helper()

	finder := refcache.NewRepoFinder(nil, nil,
		repoCache{1: {ID: 1, Identifier: "r1", Path: "space/r1"}}, nil, storecache.Evictor[*types.RepositoryCore]{})
	svc := insights.NewService(insights.Config{}, st.ProjectStore(), st.IssueStore(), st.EventStore())
	ctrl := ctrlinsights.NewController(authztest.AllowAuthorizer{}, finder, urlStub{}, svc, audit.New())

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			s := &auth.Session{Principal: types.Principal{ID: 1, UID: "alice"}}
			next.ServeHTTP(w, req.WithContext(request.WithAuthSession(req.Context(), s)))
		})
	})
	r.Route("/repos/{repo_ref}/insights", func(r chi.Router) {
		r.Post("/projects", HandleCreateProject(ctrl))
		r.Route("/projects/{project_id}", func(r chi.Router) {
			r.Patch("/", HandleUpdateProject(ctrl))
			r.Delete("/", HandleDeleteProject(ctrl))
			r.Post("/rotate-key", HandleRotateProjectKey(ctrl))
		})
		r.Get("/issues", HandleListIssues(ctrl))
		r.Route("/issues/{issue_id}", func(r chi.Router) {
			r.Get("/", HandleFindIssue(ctrl))
			r.Patch("/", HandleUpdateIssue(ctrl))
			r.Get("/events", HandleListIssueEvents(ctrl))
		})
	})
	return r
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func seed(st *insightstest.Store) {
	st.Projects[1] = &types.InsightsProject{ID: 1, RepoID: 1, Identifier: "api", PublicKey: "k1", Enabled: true}
	for i := int64(1); i <= 3; i++ {
		st.Issues[i] = &types.InsightsIssue{ID: i, ProjectID: 1, RepoID: 1, LastSeen: i * 100, TimesSeen: 4 - i,
			Status: enum.InsightsIssueStatusUnresolved, Level: "error", Title: "boom"}
	}
	st.Issues[9] = &types.InsightsIssue{ID: 9, ProjectID: 5, RepoID: 2, Status: enum.InsightsIssueStatusUnresolved}
	for i := int64(1); i <= 3; i++ {
		st.Events[i] = &types.InsightsEvent{ID: i, IssueID: 1, ProjectID: 1, Timestamp: i, Payload: `{"a":1}`}
	}
}

func TestProjectEndpoints(t *testing.T) {
	st := insightstest.New()
	seed(st)
	h := newRouter(t, st)

	rec := do(h, http.MethodPatch, "/repos/1/insights/projects/1", `{"enabled":false,"sample_rate":0.1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	var p types.InsightsProjectWithDSN
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Enabled || p.SampleRate != 0.1 {
		t.Fatalf("patch body %s (%v)", rec.Body, err)
	}

	rec = do(h, http.MethodPost, "/repos/1/insights/projects/1/rotate-key", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate: %d %s", rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.PublicKey == "k1" ||
		!strings.Contains(p.DSN, p.PublicKey) {
		t.Fatalf("rotate body %s (%v)", rec.Body, err)
	}

	if rec = do(h, http.MethodPatch, "/repos/1/insights/projects/1", `{"sample_rate":2}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid sample rate: %d", rec.Code)
	}
	if rec = do(h, http.MethodPatch, "/repos/1/insights/projects/1", `{bad`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid body: %d", rec.Code)
	}
	if rec = do(h, http.MethodPatch, "/repos/1/insights/projects/abc", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid project id: %d", rec.Code)
	}
	if rec = do(h, http.MethodPost, "/repos/1/insights/projects/77/rotate-key", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown project: %d", rec.Code)
	}

	if rec = do(h, http.MethodDelete, "/repos/1/insights/projects/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if _, ok := st.Projects[1]; ok {
		t.Error("project not deleted")
	}
}

func TestIssueEndpoints(t *testing.T) {
	st := insightstest.New()
	seed(st)
	h := newRouter(t, st)

	rec := do(h, http.MethodGet, "/repos/1/insights/issues?limit=2&sort=times_seen&order=asc&status=unresolved,ignored", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var issues []*types.InsightsIssue
	if err := json.Unmarshal(rec.Body.Bytes(), &issues); err != nil || len(issues) != 2 || issues[0].ID != 3 {
		t.Fatalf("list body %s (%v)", rec.Body, err)
	}
	if rec.Header().Get("x-total") != "3" || rec.Header().Get("x-per-page") != "2" ||
		rec.Header().Get("x-total-pages") != "2" {
		t.Errorf("pagination headers: %v", rec.Header())
	}

	if rec = do(h, http.MethodGet, "/repos/1/insights/issues?status=bogus", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid status: %d", rec.Code)
	}
	if rec = do(h, http.MethodGet, "/repos/1/insights/issues?project=x", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid project: %d", rec.Code)
	}

	if rec = do(h, http.MethodGet, "/repos/1/insights/issues/2", ""); rec.Code != http.StatusOK {
		t.Errorf("find: %d", rec.Code)
	}
	if rec = do(h, http.MethodGet, "/repos/1/insights/issues/9", ""); rec.Code != http.StatusNotFound {
		t.Errorf("issue of other repo: %d", rec.Code)
	}

	rec = do(h, http.MethodGet, "/repos/1/insights/issues/1/events?limit=2", "")
	var events []*types.InsightsEvent
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil || rec.Code != http.StatusOK ||
		len(events) != 2 || events[0].Timestamp != 3 || events[0].Payload != `{"a":1}` {
		t.Fatalf("events: %d %s (%v)", rec.Code, rec.Body, err)
	}
	if rec.Header().Get("x-total") != "3" {
		t.Errorf("events pagination headers: %v", rec.Header())
	}

	rec = do(h, http.MethodPatch, "/repos/1/insights/issues/1", `{"status":"resolved","resolved_in_release":"2.0.0"}`)
	var issue types.InsightsIssue
	if err := json.Unmarshal(rec.Body.Bytes(), &issue); err != nil || rec.Code != http.StatusOK ||
		issue.Status != enum.InsightsIssueStatusResolved || issue.ResolvedInRelease != "2.0.0" {
		t.Fatalf("patch: %d %s (%v)", rec.Code, rec.Body, err)
	}
	if rec = do(h, http.MethodPatch, "/repos/1/insights/issues/1", `{"status":"regressed"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid status: %d", rec.Code)
	}
}
