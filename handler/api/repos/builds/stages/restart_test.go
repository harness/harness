// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package stages

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/errors"
	"github.com/drone/drone/mock"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestRestart(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}
	mockStage := &core.Stage{
		ID:        222,
		BuildID:   111,
		Number:    1,
		Name:      "stage1",
		Status:    core.StatusFailing,
		DependsOn: []string{},
	}
	mockStep := &core.Step{
		ID:      333,
		StageID: 222,
		Number:  1,
		Name:    "step1",
		Status:  core.StatusFailing,
	}
	downstreamStage := &core.Stage{
		ID:        444,
		BuildID:   111,
		Number:    2,
		Name:      "stage2",
		Status:    core.StatusSkipped,
		DependsOn: []string{"stage1"},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(nil)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return([]*core.Stage{mockStage, downstreamStage}, nil)
	stages.EXPECT().Update(gomock.Any(), downstreamStage).Return(nil)

	steps := mock.NewMockStepStore(controller)
	steps.EXPECT().List(gomock.Any(), mockStage.ID).Return([]*core.Step{mockStep}, nil)
	steps.EXPECT().Update(gomock.Any(), mockStep).Return(nil)

	logs := mock.NewMockLogStore(controller)
	logs.EXPECT().Delete(gomock.Any(), mockStep.ID).Return(nil)

	sched := mock.NewMockScheduler(controller)
	sched.EXPECT().Schedule(gomock.Any(), mockStage).Return(nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, steps, sched, logs)(w, r)
	if got, want := w.Code, 204; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRestart_InvalidBuildNumber(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "XLII")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(nil, nil, nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got := new(errors.Error)
	json.NewDecoder(w.Body).Decode(got)
	if got.Message == "" {
		t.Errorf("Want error message, got empty")
	}
}

func TestRestart_InvalidStageNumber(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "two")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRestart_RepoNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, nil, nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRestart_BuildNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRestart_BuildBlocked(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusBlocked,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got := new(errors.Error)
	json.NewDecoder(w.Body).Decode(got)
	wantMsg := "cannot start a blocked build"
	if got.Message != wantMsg {
		t.Errorf("Want message %q, got %q", wantMsg, got.Message)
	}
}

func TestRestart_BuildDeclined(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusDeclined,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got := new(errors.Error)
	json.NewDecoder(w.Body).Decode(got)
	wantMsg := "cannot start a declined build"
	if got.Message != wantMsg {
		t.Errorf("Want message %q, got %q", wantMsg, got.Message)
	}
}

func TestRestart_StageNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, 1).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, nil, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf("Diff: %s", diff)
	}
}

func TestRestart_StageNotRestartable(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 1,
		Name:   "stage1",
		Status: core.StatusPending,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got := new(errors.Error)
	json.NewDecoder(w.Body).Decode(got)
	if got.Message == "" {
		t.Errorf("Want error message about cannot restart, got empty")
	}
}

func TestRestart_StepsListError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 1,
		Name:   "stage1",
		Status: core.StatusFailing,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)

	steps := mock.NewMockStepStore(controller)
	steps.EXPECT().List(gomock.Any(), mockStage.ID).Return(nil, errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, steps, nil, nil)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRestart_ScheduleError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 1,
		Name:   "stage1",
		Status: core.StatusFailing,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(nil)

	steps := mock.NewMockStepStore(controller)
	steps.EXPECT().List(gomock.Any(), mockStage.ID).Return([]*core.Step{}, nil)

	logs := mock.NewMockLogStore(controller)

	sched := mock.NewMockScheduler(controller)
	sched.EXPECT().Schedule(gomock.Any(), mockStage).Return(errors.ErrNotFound)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, steps, sched, logs)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRestart_DownstreamResetToWaiting(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}
	mockStage := &core.Stage{
		ID:        222,
		BuildID:   111,
		Number:    1,
		Name:      "stage1",
		Status:    core.StatusFailing,
		DependsOn: []string{},
	}
	downstreamStage := &core.Stage{
		ID:        444,
		BuildID:   111,
		Number:    2,
		Name:      "stage2",
		Status:    core.StatusSkipped,
		DependsOn: []string{"stage1"},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(nil)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return([]*core.Stage{mockStage, downstreamStage}, nil)
	stages.EXPECT().Update(gomock.Any(), downstreamStage).Return(nil).Do(func(_ interface{}, s interface{}) {
		st := s.(*core.Stage)
		if st.Status != core.StatusWaiting {
			t.Errorf("Want downstream stage status Waiting, got %q", st.Status)
		}
		if st.Number != 2 {
			t.Errorf("Want downstream stage number 2, got %d", st.Number)
		}
	})

	steps := mock.NewMockStepStore(controller)
	steps.EXPECT().List(gomock.Any(), mockStage.ID).Return([]*core.Step{}, nil)

	logs := mock.NewMockLogStore(controller)

	sched := mock.NewMockScheduler(controller)
	sched.EXPECT().Schedule(gomock.Any(), mockStage).Return(nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, steps, sched, logs)(w, r)
	if got, want := w.Code, 204; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

func TestRestart_NoDownstream(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		ID:        1,
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusFailing,
	}
	mockStage := &core.Stage{
		ID:        222,
		BuildID:   111,
		Number:    1,
		Name:      "stage1",
		Status:    core.StatusFailing,
		DependsOn: []string{},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(nil)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return([]*core.Stage{mockStage}, nil)

	steps := mock.NewMockStepStore(controller)
	steps.EXPECT().List(gomock.Any(), mockStage.ID).Return([]*core.Step{}, nil)

	logs := mock.NewMockLogStore(controller)

	sched := mock.NewMockScheduler(controller)
	sched.EXPECT().Schedule(gomock.Any(), mockStage).Return(nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleRestart(repos, builds, stages, steps, sched, logs)(w, r)
	if got, want := w.Code, 204; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}
