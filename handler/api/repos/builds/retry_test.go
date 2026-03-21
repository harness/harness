// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package builds

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/errors"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/mock"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestRetry(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkBuild := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		if got, want := hook.Trigger, mockUser.Login; got != want {
			t.Errorf("Want Trigger By %s, got %s", want, got)
		}
		if got, want := hook.Event, mockBuild.Event; got != want {
			t.Errorf("Want Build Event %s, got %s", want, got)
		}
		if got, want := hook.Link, mockBuild.Link; got != want {
			t.Errorf("Want Build Link %s, got %s", want, got)
		}
		if got, want := hook.Message, mockBuild.Message; got != want {
			t.Errorf("Want Build Message %s, got %s", want, got)
		}
		if got, want := hook.Before, mockBuild.Before; got != want {
			t.Errorf("Want Build Before %s, got %s", want, got)
		}
		if got, want := hook.After, mockBuild.After; got != want {
			t.Errorf("Want Build After %s, got %s", want, got)
		}
		if got, want := hook.Ref, mockBuild.Ref; got != want {
			t.Errorf("Want Build Ref %s, got %s", want, got)
		}
		if got, want := hook.Source, mockBuild.Source; got != want {
			t.Errorf("Want Build Source %s, got %s", want, got)
		}
		if got, want := hook.Target, mockBuild.Target; got != want {
			t.Errorf("Want Build Target %s, got %s", want, got)
		}
		if got, want := hook.Author, mockBuild.Author; got != want {
			t.Errorf("Want Build Author %s, got %s", want, got)
		}
		if got, want := hook.AuthorName, mockBuild.AuthorName; got != want {
			t.Errorf("Want Build AuthorName %s, got %s", want, got)
		}
		if got, want := hook.AuthorEmail, mockBuild.AuthorEmail; got != want {
			t.Errorf("Want Build AuthorEmail %s, got %s", want, got)
		}
		if got, want := hook.AuthorAvatar, mockBuild.AuthorAvatar; got != want {
			t.Errorf("Want Build AuthorAvatar %s, got %s", want, got)
		}
		if got, want := hook.Sender, mockBuild.Sender; got != want {
			t.Errorf("Want Build Sender %s, got %s", want, got)
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkBuild)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(core.Build), mockBuild
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRetry_InvalidBuildNumber(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "XLII")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), &errors.Error{
		Message: `strconv.ParseInt: parsing "XLII": invalid syntax`,
	}
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRetry_RepoNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, nil, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRetry_BuildNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRetry_TriggerError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, triggerer)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRetry_WithStage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkHook := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		if got, want := hook.Stages, []string{"build"}; len(got) != len(want) || got[0] != want[0] {
			t.Errorf("Want hook.Stages %v, got %v", want, got)
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkHook)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_WithStageAndCascade(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	prevStages := []*core.Stage{
		{Name: "build", DependsOn: nil},
		{Name: "test", DependsOn: []string{"build"}},
		{Name: "deploy", DependsOn: []string{"test"}},
	}

	checkHook := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		got := make([]string, len(hook.Stages))
		copy(got, hook.Stages)
		sort.Strings(got)
		want := []string{"build", "deploy", "test"}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("hook.Stages mismatch (-got +want):\n%s", diff)
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(prevStages, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkHook)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build&cascade=true", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, stages, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_WithStageCascade_StageListError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build&cascade=true", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, stages, nil)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_StageAndCascadeExcludedFromParams(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkHook := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		if _, ok := hook.Params["stage"]; ok {
			t.Error("Expected 'stage' to be excluded from hook.Params")
		}
		if _, ok := hook.Params["cascade"]; ok {
			t.Error("Expected 'cascade' to be excluded from hook.Params")
		}
		if got, want := hook.Params["custom"], "value"; got != want {
			t.Errorf("Want hook.Params[custom] = %s, got %s", want, got)
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return([]*core.Stage{
		{Name: "build"},
	}, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkHook)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build&cascade=true&custom=value", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, stages, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_BlockedBuild_WithStage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	blockedBuild := &core.Build{
		ID:     mockBuild.ID,
		Number: mockBuild.Number,
		RepoID: mockBuild.RepoID,
		Status: core.StatusBlocked,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(blockedBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_DeclinedBuild_WithStage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	declinedBuild := &core.Build{
		ID:     mockBuild.ID,
		Number: mockBuild.Number,
		RepoID: mockBuild.RepoID,
		Status: core.StatusDeclined,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(declinedBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build&cascade=true", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_WithStageAndCascadeFalse(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkHook := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		if got, want := hook.Stages, []string{"build"}; len(got) != len(want) || got[0] != want[0] {
			t.Errorf("Want hook.Stages %v, got %v", want, got)
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkHook)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=build&cascade=false", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_WithStageWhitespace(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkHook := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		if got, want := hook.Stages, []string{"build"}; len(got) != len(want) || got[0] != want[0] {
			t.Errorf("Want hook.Stages %v, got %v", want, got)
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkHook)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=+build+", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, nil, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRetry_WithStageCascade_StageNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	prevStages := []*core.Stage{
		{Name: "build", DependsOn: nil},
		{Name: "test", DependsOn: []string{"build"}},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(prevStages, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?stage=nonexistent&cascade=true", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleRetry(repos, builds, stages, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}
