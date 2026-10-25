// Copyright 2026 Harness, Inc.
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

package aiagent

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeAuthorizer struct {
	allow   func(res *types.Resource, perm enum.Permission) bool
	checked []enum.Permission
}

func (a *fakeAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, res *types.Resource, perm enum.Permission,
) (bool, error) {
	a.checked = append(a.checked, perm)
	return a.allow == nil || a.allow(res, perm), nil
}

func (a *fakeAuthorizer) CheckAll(
	ctx context.Context, s *auth.Session, checks ...types.PermissionCheck,
) (bool, error) {
	for _, c := range checks {
		if ok, _ := a.Check(ctx, s, &c.Scope, &c.Resource, c.Permission); !ok {
			return false, nil
		}
	}
	return true, nil
}

func (a *fakeAuthorizer) CheckMany(
	ctx context.Context, s *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	out := make([]bool, len(checks))
	for i, c := range checks {
		out[i], _ = a.Check(ctx, s, &c.Scope, &c.Resource, c.Permission)
	}
	return out, nil
}

type fakeSpaces struct{}

func (fakeSpaces) FindByRef(context.Context, string) (*types.SpaceCore, error) {
	return &types.SpaceCore{ID: 10, Path: "space"}, nil
}

type fakeRepos struct {
	byID map[int64]*types.RepositoryCore
}

func (f fakeRepos) FindByID(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if r, ok := f.byID[id]; ok {
		return r, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type fakeRepoLister struct{}

func (fakeRepoLister) List(_ context.Context, _ int64, f *types.RepoFilter) ([]*types.Repository, error) {
	if f.Page > 1 {
		return nil, nil
	}
	return []*types.Repository{{ID: 1}, {ID: 2}}, nil
}

type fakeTasks struct {
	store.TaskStore
	byID     map[int64]*types.Task
	waiting  []*types.Task
	listed   []int64
	listedBy *types.TaskFilter
}

func (f *fakeTasks) ListByRepos(_ context.Context, repoIDs []int64, filter *types.TaskFilter) ([]*types.Task, error) {
	f.listed, f.listedBy = repoIDs, filter
	var out []*types.Task
	for _, t := range f.byID {
		for _, id := range repoIDs {
			if t.RepoID == id && t.State != enum.TaskStateWaitingQuota {
				out = append(out, t)
			}
		}
	}
	return out, nil
}

func (f *fakeTasks) CountByRepos(_ context.Context, repoIDs []int64, _ *types.TaskFilter) (int64, error) {
	var n int64
	for _, t := range f.waiting {
		for _, id := range repoIDs {
			if t.RepoID == id {
				n++
			}
		}
	}
	return n, nil
}

func (f *fakeTasks) ListWaitingQuota(context.Context, int) ([]*types.Task, error) {
	return f.waiting, nil
}

func (f *fakeTasks) Find(_ context.Context, id int64) (*types.Task, error) {
	if t, ok := f.byID[id]; ok {
		return t, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type fakeActivities struct {
	store.TaskActivityStore
	runs []*types.TaskActivity
}

func (f fakeActivities) ListLatestByKindForRepos(
	_ context.Context, _ []int64, kind enum.TaskActivityKind, limit int,
) ([]*types.TaskActivity, error) {
	if kind != enum.TaskActivityKindAgentRun || limit != maxRunsShown {
		return nil, nil
	}
	return f.runs, nil
}

type fakeAITasks struct{ store.AITaskStore }

func (fakeAITasks) ListByTaskIDs(context.Context, []int64) ([]*types.AITask, error) {
	id := int64(100)
	return []*types.AITask{
		{TaskID: &id, OutputMetadata: json.RawMessage(`{"agent_model":"old/model"}`)},
		{TaskID: &id, OutputMetadata: json.RawMessage(`{"agent_model":"new/model"}`)},
	}, nil
}

type fakeModels struct{ store.AIModelStore }

func (fakeModels) List(context.Context, types.AIModelFilter) ([]*types.AIModel, error) {
	return []*types.AIModel{
		{ID: "b/model", SuccessRateByTask: map[string]float64{"fix_sast": 0.5, "fix_sast#n": 4, "fix_error": 1, "fix_error#n": 2}},
		{ID: "a/model", SuccessRateByTask: map[string]float64{"fix_error": 0.75, "fix_error#n": 8}},
		{ID: "c/model", SuccessRateByTask: map[string]float64{}},
	}, nil
}

type fakeUsage struct {
	store.AIUsageStore
	repos []int64
	since int64
}

func (f *fakeUsage) SumRequestsSince(_ context.Context, spaceID int64, repoIDs []int64, since int64) (int, error) {
	f.repos, f.since = repoIDs, since
	if spaceID != 10 {
		return 0, nil
	}
	return 17, nil
}

type fakeQuota struct{ enabled bool }

func (q fakeQuota) Enabled() bool { return q.enabled }

func (fakeQuota) Snapshot() aigateway.QuotaSnapshot {
	return aigateway.QuotaSnapshot{
		Day: "2026-03-04", Limit: 50, Used: 12,
		UsedClass: map[aigateway.Class]int{aigateway.ClassAgent: 9, aigateway.ClassTriage: 3},
	}
}

func (fakeQuota) Remaining(class aigateway.Class) int {
	if class == aigateway.ClassAgent {
		return 21
	}
	return 4
}

type env struct {
	ctrl    *Controller
	authz   *fakeAuthorizer
	tasks   *fakeTasks
	usage   *fakeUsage
	session *auth.Session
}

func newEnv(enabled bool) *env {
	repo1 := &types.RepositoryCore{ID: 1, Path: "space/one", Identifier: "one"}
	repo2 := &types.RepositoryCore{ID: 2, Path: "space/two", Identifier: "two"}
	e := &env{
		authz: &fakeAuthorizer{},
		tasks: &fakeTasks{
			byID: map[int64]*types.Task{
				100: {ID: 100, RepoID: 1, Number: 1, Title: "active", State: enum.TaskStateInProgress},
				101: {ID: 101, RepoID: 2, Number: 2, Title: "other repo", State: enum.TaskStateInReview},
				102: {ID: 102, RepoID: 1, Number: 3, Title: "waiting", State: enum.TaskStateWaitingQuota},
			},
		},
		usage:   &fakeUsage{},
		session: &auth.Session{Principal: types.Principal{ID: 7, Type: enum.PrincipalTypeUser}},
	}
	e.tasks.waiting = []*types.Task{
		{ID: 200, RepoID: 99, Number: 9, Title: "foreign", State: enum.TaskStateWaitingQuota},
		e.tasks.byID[102],
	}
	acts := fakeActivities{runs: []*types.TaskActivity{
		{TaskID: 100, Kind: enum.TaskActivityKindAgentRun, Created: 30,
			Payload: json.RawMessage(`{"model":"new/model","requests_used":12,"outcome":"pr_opened"}`)},
		{TaskID: 12345, Kind: enum.TaskActivityKindAgentRun, Created: 20, Payload: json.RawMessage(`{"model":"gone"}`)},
		{TaskID: 101, Kind: enum.TaskActivityKindAgentRun, Created: 10, Payload: json.RawMessage(`not json`)},
	}}
	e.ctrl = NewController(
		e.authz, fakeSpaces{},
		fakeRepos{byID: map[int64]*types.RepositoryCore{1: repo1, 2: repo2}},
		fakeRepoLister{}, e.tasks, acts, fakeAITasks{}, fakeModels{}, e.usage, fakeQuota{enabled: enabled})
	return e
}

func statusOf(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return usererror.Translate(context.Background(), err).Status
}

func TestPanelAggregation(t *testing.T) {
	e := newEnv(true)

	p, err := e.ctrl.Panel(context.Background(), e.session, "space")
	if err != nil {
		t.Fatalf("Panel() error = %v", err)
	}

	if !p.Enabled || p.Quota.Limit != 50 || p.Quota.Used != 12 || p.Quota.Day != "2026-03-04" {
		t.Errorf("quota = %+v", p.Quota)
	}
	if p.Quota.UsedByClass["agent"] != 9 || p.Quota.RemainingByClass["agent"] != 21 ||
		p.Quota.RemainingByClass["manual"] != 4 {
		t.Errorf("quota by class = %+v / %+v", p.Quota.UsedByClass, p.Quota.RemainingByClass)
	}
	if p.Quota.SpaceUsedToday != 17 {
		t.Errorf("space used = %d, want 17", p.Quota.SpaceUsedToday)
	}
	if want := int64(1772582400000); e.usage.since != want { // 2026-03-04T00:00:00Z
		t.Errorf("usage since = %d, want %d", e.usage.since, want)
	}
	if len(e.usage.repos) != 2 {
		t.Errorf("usage repos = %v", e.usage.repos)
	}

	f := e.tasks.listedBy
	if f == nil || f.IsAgent == nil || !*f.IsAgent || len(f.States) != 2 ||
		f.States[0] != enum.TaskStateInProgress || f.States[1] != enum.TaskStateInReview {
		t.Errorf("active tasks filter = %+v", f)
	}
	if len(p.ActiveTasks) != 2 {
		t.Fatalf("active tasks = %+v", p.ActiveTasks)
	}
	for _, at := range p.ActiveTasks {
		switch at.Number {
		case 1:
			if at.RepoPath != "space/one" || at.Model != "new/model" {
				t.Errorf("active task 1 = %+v, want latest model new/model", at)
			}
		case 2:
			if at.RepoPath != "space/two" || at.Model != "" {
				t.Errorf("active task 2 = %+v", at)
			}
		}
	}

	if p.WaitingQuota.Count != 1 || len(p.WaitingQuota.Tasks) != 1 || p.WaitingQuota.Tasks[0].Number != 3 {
		t.Errorf("waiting = %+v, want only the task of the space", p.WaitingQuota)
	}

	if len(p.ModelSuccess) != 2 || p.ModelSuccess[0].Model != "a/model" || p.ModelSuccess[1].Model != "b/model" {
		t.Fatalf("model success = %+v", p.ModelSuccess)
	}
	b := p.ModelSuccess[1].Kinds
	if len(b) != 2 || b[0].Kind != "fix_error" || b[0].Samples != 2 || b[0].Rate != 1 ||
		b[1].Kind != "fix_sast" || b[1].Samples != 4 || b[1].Rate != 0.5 {
		t.Errorf("kinds of b/model = %+v", b)
	}

	if len(p.LatestRuns) != 1 {
		t.Fatalf("runs = %+v, want only the valid one of a known task", p.LatestRuns)
	}
	r := p.LatestRuns[0]
	if r.Model != "new/model" || r.Outcome != "pr_opened" || r.RequestsUsed != 12 ||
		r.TaskNumber != 1 || r.RepoPath != "space/one" {
		t.Errorf("run = %+v", r)
	}
}

func TestPanelWaitingQueueLimit(t *testing.T) {
	e := newEnv(true)
	e.tasks.waiting = nil
	for i := range 30 {
		e.tasks.waiting = append(e.tasks.waiting, &types.Task{
			ID: int64(300 + i), RepoID: 1, Number: int64(i + 1), State: enum.TaskStateWaitingQuota,
		})
	}

	p, err := e.ctrl.Panel(context.Background(), e.session, "space")
	if err != nil {
		t.Fatalf("Panel() error = %v", err)
	}
	if p.WaitingQuota.Count != 30 || len(p.WaitingQuota.Tasks) != 20 || p.WaitingQuota.Tasks[0].Number != 1 {
		t.Errorf("waiting count=%d len=%d", p.WaitingQuota.Count, len(p.WaitingQuota.Tasks))
	}
}

func TestPanelGatewayDisabled(t *testing.T) {
	p, err := newEnv(false).ctrl.Panel(context.Background(), newEnv(false).session, "space")
	if err != nil {
		t.Fatalf("Panel() error = %v", err)
	}
	if p.Enabled || p.Quota.Limit != 0 || len(p.Quota.UsedByClass) != 0 || p.Quota.SpaceUsedToday != 17 {
		t.Errorf("panel = %+v", p)
	}
}

func TestPanelPermissions(t *testing.T) {
	e := newEnv(true)
	e.authz.allow = func(*types.Resource, enum.Permission) bool { return false }
	if _, err := e.ctrl.Panel(context.Background(), e.session, "space"); statusOf(err) != http.StatusForbidden {
		t.Errorf("status = %d, want 403", statusOf(err))
	}

	e = newEnv(true)
	if _, err := e.ctrl.Panel(context.Background(), e.session, "space"); err != nil {
		t.Fatalf("Panel() error = %v", err)
	}
	if e.authz.checked[0] != enum.PermissionSpaceView {
		t.Errorf("first permission = %v, want space_view", e.authz.checked[0])
	}
}

func TestPanelOnlyViewableRepos(t *testing.T) {
	e := newEnv(true)
	e.authz.allow = func(res *types.Resource, perm enum.Permission) bool {
		return perm == enum.PermissionSpaceView || res.Identifier != "two"
	}

	p, err := e.ctrl.Panel(context.Background(), e.session, "space")
	if err != nil {
		t.Fatalf("Panel() error = %v", err)
	}
	if len(e.tasks.listed) != 1 || e.tasks.listed[0] != 1 {
		t.Errorf("listed repos = %v, want [1]", e.tasks.listed)
	}
	if len(p.ActiveTasks) != 1 || p.ActiveTasks[0].Number != 1 {
		t.Errorf("active = %+v", p.ActiveTasks)
	}
}
