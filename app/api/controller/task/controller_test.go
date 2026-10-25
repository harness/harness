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

package task

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	taskservice "github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// recordingAuthorizer grants the permissions in allow and records every check.
type recordingAuthorizer struct {
	allow   func(resource *types.Resource, scope *types.Scope, permission enum.Permission) bool
	checked []enum.Permission
}

func (a *recordingAuthorizer) Check(
	_ context.Context, _ *auth.Session, scope *types.Scope, resource *types.Resource, permission enum.Permission,
) (bool, error) {
	a.checked = append(a.checked, permission)
	if a.allow == nil {
		return true, nil
	}
	return a.allow(resource, scope, permission), nil
}

func (a *recordingAuthorizer) CheckAll(
	ctx context.Context, s *auth.Session, checks ...types.PermissionCheck,
) (bool, error) {
	for _, c := range checks {
		if ok, _ := a.Check(ctx, s, &c.Scope, &c.Resource, c.Permission); !ok {
			return false, nil
		}
	}
	return true, nil
}

func (a *recordingAuthorizer) CheckMany(
	ctx context.Context, s *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	out := make([]bool, len(checks))
	for i, c := range checks {
		out[i], _ = a.Check(ctx, s, &c.Scope, &c.Resource, c.Permission)
	}
	return out, nil
}

type fakeRepos struct {
	byRef map[string]*types.RepositoryCore
	byID  map[int64]*types.RepositoryCore
}

func (f fakeRepos) FindByRef(_ context.Context, ref string) (*types.RepositoryCore, error) {
	if r, ok := f.byRef[ref]; ok {
		return r, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f fakeRepos) FindByID(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if r, ok := f.byID[id]; ok {
		return r, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type fakeSpaces struct{ space *types.SpaceCore }

func (f fakeSpaces) FindByRef(context.Context, string) (*types.SpaceCore, error) {
	return f.space, nil
}

type fakeRepoLister struct{ repos []*types.Repository }

func (f fakeRepoLister) List(
	_ context.Context, _ int64, filter *types.RepoFilter,
) ([]*types.Repository, error) {
	if filter.Page > 1 {
		return nil, nil
	}
	return f.repos, nil
}

type fakeTaskStore struct {
	store.TaskStore
	tasks     map[int64]*types.Task // by number, repo 1 only
	listedFor []int64
}

func (f *fakeTaskStore) FindByNumber(_ context.Context, repoID, number int64) (*types.Task, error) {
	if t, ok := f.tasks[number]; ok && t.RepoID == repoID {
		return t, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f *fakeTaskStore) ListByRepos(_ context.Context, repoIDs []int64, _ *types.TaskFilter) ([]*types.Task, error) {
	f.listedFor = repoIDs
	var out []*types.Task
	for _, t := range f.tasks {
		for _, id := range repoIDs {
			if t.RepoID == id {
				out = append(out, t)
			}
		}
	}
	return out, nil
}

func (f *fakeTaskStore) CountByRepos(_ context.Context, _ []int64, _ *types.TaskFilter) (int64, error) {
	return int64(len(f.tasks)), nil
}

type fakeActivities struct{ store.TaskActivityStore }

func (fakeActivities) List(context.Context, int64, types.Pagination) ([]*types.TaskActivity, error) {
	return nil, nil
}

type fakeLinks struct{ store.TaskLinkStore }

func (fakeLinks) ListByTask(context.Context, int64) ([]*types.TaskLink, error) { return nil, nil }

type fakeService struct {
	createIn   taskservice.CreateInput
	err        error
	calledWith int64
}

func (f *fakeService) Create(
	_ context.Context, _ int64, in taskservice.CreateInput,
) (*types.Task, bool, error) {
	f.createIn = in
	if f.err != nil {
		return nil, false, f.err
	}
	return &types.Task{ID: 100, RepoID: in.RepoID, Number: 9, Title: in.Title}, true, nil
}

func (f *fakeService) Update(_ context.Context, _ int64, id int64, _ taskservice.UpdateInput) (*types.Task, error) {
	f.calledWith = id
	return &types.Task{ID: id}, f.err
}

func (f *fakeService) Transition(
	_ context.Context, _ int64, id int64, _ enum.TaskState, _ string,
) (*types.Task, error) {
	f.calledWith = id
	return nil, f.err
}

func (f *fakeService) Resume(_ context.Context, _ int64, id int64, _ string) (*types.Task, error) {
	f.calledWith = id
	return &types.Task{ID: id}, f.err
}

func (f *fakeService) AddComment(_ context.Context, _ int64, id int64, _ string) error {
	f.calledWith = id
	return f.err
}

type testEnv struct {
	ctrl    *Controller
	authz   *recordingAuthorizer
	service *fakeService
	tasks   *fakeTaskStore
	session *auth.Session
}

func newTestEnv() *testEnv {
	repo1 := &types.RepositoryCore{ID: 1, Path: "space/one", Identifier: "one"}
	repo2 := &types.RepositoryCore{ID: 2, Path: "space/two", Identifier: "two"}
	repos := fakeRepos{
		byRef: map[string]*types.RepositoryCore{"space/one": repo1, "space/two": repo2},
		byID:  map[int64]*types.RepositoryCore{1: repo1, 2: repo2},
	}
	e := &testEnv{
		authz:   &recordingAuthorizer{},
		service: &fakeService{},
		tasks: &fakeTaskStore{tasks: map[int64]*types.Task{
			5: {ID: 50, RepoID: 1, Number: 5, Title: "five"},
			6: {ID: 60, RepoID: 2, Number: 6, Title: "six"},
		}},
		session: &auth.Session{Principal: types.Principal{ID: 7, Type: enum.PrincipalTypeUser}},
	}
	e.ctrl = NewController(
		e.authz, repos,
		fakeSpaces{space: &types.SpaceCore{ID: 10, Path: "space"}},
		fakeRepoLister{repos: []*types.Repository{{ID: 1}, {ID: 2}}},
		e.tasks, fakeActivities{}, fakeLinks{}, e.service)
	return e
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return http.StatusOK
	}
	return usererror.Translate(context.Background(), err).Status
}

func TestPermissions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		want enum.Permission
		call func(e *testEnv) error
	}{
		{"list", enum.PermissionRepoView, func(e *testEnv) error {
			_, _, err := e.ctrl.List(ctx, e.session, "space/one", &types.TaskFilter{Page: 1, Size: 10})
			return err
		}},
		{"find", enum.PermissionRepoView, func(e *testEnv) error {
			_, err := e.ctrl.Find(ctx, e.session, "space/one", 5)
			return err
		}},
		{"activities", enum.PermissionRepoView, func(e *testEnv) error {
			_, err := e.ctrl.ListActivities(ctx, e.session, "space/one", 5, types.Pagination{})
			return err
		}},
		{"links", enum.PermissionRepoView, func(e *testEnv) error {
			_, err := e.ctrl.ListLinks(ctx, e.session, "space/one", 5)
			return err
		}},
		{"create", enum.PermissionRepoPush, func(e *testEnv) error {
			_, err := e.ctrl.Create(ctx, e.session, "space/one", &CreateInput{Title: "t"})
			return err
		}},
		{"update", enum.PermissionRepoPush, func(e *testEnv) error {
			_, err := e.ctrl.Update(ctx, e.session, "space/one", 5, &UpdateInput{})
			return err
		}},
		{"state", enum.PermissionRepoPush, func(e *testEnv) error {
			_, err := e.ctrl.State(ctx, e.session, "space/one", 5, &StateInput{State: enum.TaskStateTriage})
			return err
		}},
		{"resume", enum.PermissionRepoPush, func(e *testEnv) error {
			_, err := e.ctrl.Resume(ctx, e.session, "space/one", 5, &ResumeInput{})
			return err
		}},
		{"comment", enum.PermissionRepoPush, func(e *testEnv) error {
			return e.ctrl.CreateComment(ctx, e.session, "space/one", 5, &CommentInput{Text: "hi"})
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv()
			_ = tc.call(e)
			if len(e.authz.checked) == 0 || e.authz.checked[0] != tc.want {
				t.Errorf("checked permissions = %v, want %q first", e.authz.checked, tc.want)
			}

			// without the permission the call is forbidden and never reaches the service
			e = newTestEnv()
			e.authz.allow = func(*types.Resource, *types.Scope, enum.Permission) bool { return false }
			if got := statusOf(t, tc.call(e)); got != http.StatusForbidden {
				t.Errorf("status without permission = %d, want 403", got)
			}
			if e.service.calledWith != 0 || e.service.createIn.Title != "" {
				t.Error("the service was called without permission")
			}
		})
	}
}

func TestViewerCannotEdit(t *testing.T) {
	e := newTestEnv()
	e.authz.allow = func(_ *types.Resource, _ *types.Scope, p enum.Permission) bool {
		return p == enum.PermissionRepoView
	}
	ctx := context.Background()

	if _, err := e.ctrl.Find(ctx, e.session, "space/one", 5); err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if _, err := e.ctrl.Update(ctx, e.session, "space/one", 5, &UpdateInput{}); statusOf(t, err) != http.StatusForbidden {
		t.Errorf("Update() error = %v, want forbidden", err)
	}
}

func TestErrorMapping(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv()

	t.Run("unknown task is 404", func(t *testing.T) {
		_, err := e.ctrl.Find(ctx, e.session, "space/one", 99)
		if got := statusOf(t, err); got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
	})

	t.Run("task of another repo is 404", func(t *testing.T) {
		_, err := e.ctrl.Find(ctx, e.session, "space/one", 6)
		if got := statusOf(t, err); got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
	})

	t.Run("unknown repo is 404", func(t *testing.T) {
		_, err := e.ctrl.Find(ctx, e.session, "space/nope", 5)
		if got := statusOf(t, err); got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
	})

	t.Run("empty repo ref is 400", func(t *testing.T) {
		_, err := e.ctrl.Find(ctx, e.session, "", 5)
		if got := statusOf(t, err); got != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", got)
		}
	})

	t.Run("invalid transition is 400", func(t *testing.T) {
		e.service.err = &taskservice.TransitionError{
			From: enum.TaskStateOpen, To: enum.TaskStateVerified, Message: "Task can't move",
		}
		_, err := e.ctrl.State(ctx, e.session, "space/one", 5, &StateInput{State: enum.TaskStateVerified})
		if got := statusOf(t, err); got != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", got)
		}
		if !errors.Is(err, taskservice.ErrInvalidTransition) {
			t.Error("the transition error was lost")
		}
		e.service.err = nil
	})

	t.Run("validation error of the service is 400", func(t *testing.T) {
		e.service.err = usererror.BadRequest("Task title is required")
		_, err := e.ctrl.Create(ctx, e.session, "space/one", &CreateInput{})
		if got := statusOf(t, err); got != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", got)
		}
		e.service.err = nil
	})

	t.Run("unknown parent task is 404", func(t *testing.T) {
		parent := int64(77)
		_, err := e.ctrl.Create(ctx, e.session, "space/one", &CreateInput{Title: "t", ParentNumber: &parent})
		if got := statusOf(t, err); got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
	})

	t.Run("anonymous is 401", func(t *testing.T) {
		e.authz.allow = func(*types.Resource, *types.Scope, enum.Permission) bool { return false }
		anon := &auth.Session{Principal: auth.AnonymousPrincipal}
		_, err := e.ctrl.Find(ctx, anon, "space/one", 5)
		if got := statusOf(t, err); got != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", got)
		}
		e.authz.allow = nil
	})
}

func TestCreateIsManualAndResolvesParent(t *testing.T) {
	e := newTestEnv()
	parent := int64(5)

	task, err := e.ctrl.Create(context.Background(), e.session, "space/one", &CreateInput{
		Title: "child", Priority: enum.TaskPriorityP1, ParentNumber: &parent,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.Number != 9 {
		t.Errorf("Number = %d, want 9", task.Number)
	}

	in := e.service.createIn
	if in.Origin != enum.TaskOriginManual || in.OriginRef != "" {
		t.Errorf("origin = %q/%q, want manual", in.Origin, in.OriginRef)
	}
	if in.RepoID != 1 || in.ParentID == nil || *in.ParentID != 50 {
		t.Errorf("repo = %d, parent = %v, want repo 1 and parent id 50", in.RepoID, in.ParentID)
	}
}

func TestListForSpaceOnlyIncludesViewableRepos(t *testing.T) {
	e := newTestEnv()
	e.authz.allow = func(res *types.Resource, _ *types.Scope, _ enum.Permission) bool {
		// view of the space scope and of repo "one" only
		return res.Identifier != "two"
	}

	list, count, err := e.ctrl.ListForSpace(
		context.Background(), e.session, "space", false, &types.TaskFilter{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("ListForSpace() error = %v", err)
	}
	if len(e.tasks.listedFor) != 1 || e.tasks.listedFor[0] != 1 {
		t.Errorf("listed repos = %v, want [1]", e.tasks.listedFor)
	}
	if len(list) != 1 || list[0].RepoPath != "space/one" || list[0].Number != 5 {
		t.Fatalf("list = %+v, want the task 5 of space/one", list)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestListForSpaceForbidden(t *testing.T) {
	e := newTestEnv()
	e.authz.allow = func(*types.Resource, *types.Scope, enum.Permission) bool { return false }

	_, _, err := e.ctrl.ListForSpace(
		context.Background(), e.session, "space", false, &types.TaskFilter{Page: 1, Size: 10})
	if got := statusOf(t, err); got != http.StatusForbidden {
		t.Errorf("status = %d, want 403", got)
	}
}
