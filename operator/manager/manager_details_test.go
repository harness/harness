// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

//go:build !oss
// +build !oss

package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
)

// testStage returns a minimal stage fixture used across tests.
func testStage() *core.Stage {
	return &core.Stage{
		ID:      1,
		BuildID: 2,
		RepoID:  3,
		Status:  core.StatusPending,
		Version: 1,
	}
}

// testBuild returns a minimal build fixture.
func testBuild() *core.Build {
	return &core.Build{
		ID:     2,
		RepoID: 3,
		Status: core.StatusPending,
	}
}

// testRepo returns a minimal repository fixture.
func testRepo() *core.Repository {
	return &core.Repository{
		ID:        3,
		UserID:    4,
		Namespace: "nytimes",
	}
}

// testUser returns a minimal user fixture.
func testUser() *core.User {
	return &core.User{ID: 4}
}

// testConfig returns a minimal config fixture.
func testConfig() *core.Config {
	return &core.Config{Data: "kind: pipeline\nname: default\n"}
}

// --- handleDetailsError ---

// TestHandleDetailsError_StageMarkedError verifies that the stage is
// persisted with StatusError when handleDetailsError is called.
func TestHandleDetailsError_StageMarkedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	inputErr := errors.New("conversion failed")

	stages := mock.NewMockStageStore(ctrl)
	builds := mock.NewMockBuildStore(ctrl)
	repos := mock.NewMockRepositoryStore(ctrl)
	users := mock.NewMockUserStore(ctrl)
	events := mock.NewMockPubsub(ctrl)
	logs := mock.NewMockLogStream(ctrl)
	webhook := mock.NewMockWebhookSender(ctrl)
	status := mock.NewMockStatusService(ctrl)
	sched := mock.NewMockScheduler(ctrl)

	// Step 1: direct stage update — must be called with StatusError
	stages.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if s.Status != core.StatusError {
				t.Errorf("expected stage status %s, got %s", core.StatusError, s.Status)
			}
			if s.Error != inputErr.Error() {
				t.Errorf("expected stage error %q, got %q", inputErr.Error(), s.Error)
			}
			if s.Stopped == 0 {
				t.Error("expected Stopped to be set")
			}
			return nil
		})

	// Step 2: AfterAll teardown — Builds.Find is the first thing teardown calls.
	// Return the build so teardown can proceed far enough to also call Stages.Update.
	build := testBuild()
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)

	repo := testRepo()
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)

	// teardown calls Steps.Update for each step (none here) then Stages.Update
	stages.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil)

	// teardown calls ListSteps to decide if build is complete
	stages.EXPECT().
		ListSteps(gomock.Any(), build.ID).
		Return([]*core.Stage{stage}, nil)

	// isBuildComplete will see stage as done (StatusError), so build teardown runs
	builds.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil)

	events.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
	webhook.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)

	user := testUser()
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	status.EXPECT().Send(gomock.Any(), user, gomock.Any()).Return(nil)

	m := &Manager{
		Builds:    builds,
		Events:    events,
		Logz:      logs,
		Repos:     repos,
		Scheduler: sched,
		Stages:    stages,
		Status:    status,
		Users:     users,
		Webhook:   webhook,
	}

	got, err := m.handleDetailsError(context.Background(), stage, inputErr)
	if got != nil {
		t.Error("expected nil context return")
	}
	if err != inputErr {
		t.Errorf("expected original error to be returned, got %v", err)
	}
}

// TestHandleDetailsError_StagePersistedEvenWhenAfterAllFails verifies that
// Stage.Update is still called even when AfterAll fails (e.g. build missing).
// This is the critical regression guard for the "build not found" scenario.
func TestHandleDetailsError_StagePersistedEvenWhenAfterAllFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	inputErr := errors.New("conversion failed")
	buildErr := errors.New("build not found")

	stages := mock.NewMockStageStore(ctrl)
	builds := mock.NewMockBuildStore(ctrl)

	// Step 1: direct stage update — MUST succeed
	stages.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if s.Status != core.StatusError {
				t.Errorf("expected stage status %s, got %s", core.StatusError, s.Status)
			}
			return nil
		})

	// Step 2: AfterAll fails because build doesn't exist
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(nil, buildErr)

	m := &Manager{
		Builds: builds,
		Stages: stages,
	}

	got, err := m.handleDetailsError(context.Background(), stage, inputErr)
	if got != nil {
		t.Error("expected nil context return")
	}
	if err != inputErr {
		t.Errorf("expected original error returned, got %v", err)
	}
}

// TestHandleDetailsError_LongErrorTruncated verifies that error messages
// longer than 500 chars are truncated before persisting.
func TestHandleDetailsError_LongErrorTruncated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	longMsg := make([]byte, 600)
	for i := range longMsg {
		longMsg[i] = 'x'
	}
	inputErr := errors.New(string(longMsg))

	builds := mock.NewMockBuildStore(ctrl)
	stages := mock.NewMockStageStore(ctrl)

	stages.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if len(s.Error) > 500 {
				t.Errorf("expected error truncated to 500, got len %d", len(s.Error))
			}
			return nil
		})

	builds.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))

	m := &Manager{
		Builds: builds,
		Stages: stages,
	}
	m.handleDetailsError(context.Background(), stage, inputErr) //nolint
}

// --- Details() error paths ---

// TestDetails_StageNotFound verifies that when the stage doesn't exist in DB,
// the function returns an error without attempting any state update.
func TestDetails_StageNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stageErr := errors.New("sql: no rows")
	stages := mock.NewMockStageStore(ctrl)
	stages.EXPECT().Find(gomock.Any(), int64(1)).Return(nil, stageErr)

	m := &Manager{Stages: stages}
	_, err := m.Details(context.Background(), 1)
	if err != stageErr {
		t.Errorf("expected stage-not-found error, got %v", err)
	}
}

// TestDetails_BuildNotFound verifies that when the build is missing,
// the stage is persisted as ERROR (Step 1) even though AfterAll fails.
func TestDetails_BuildNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	buildErr := errors.New("build not found")

	stages := mock.NewMockStageStore(ctrl)
	builds := mock.NewMockBuildStore(ctrl)

	// Details: find the stage successfully
	stages.EXPECT().Find(gomock.Any(), stage.ID).Return(stage, nil)
	// Details: build lookup fails
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(nil, buildErr)

	// handleDetailsError Step 1: stage must be updated to ERROR
	stages.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if s.Status != core.StatusError {
				t.Errorf("want StatusError, got %s", s.Status)
			}
			return nil
		})

	// handleDetailsError Step 2 (AfterAll): build lookup fails again → teardown aborts
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(nil, buildErr)

	m := &Manager{Builds: builds, Stages: stages}
	_, err := m.Details(context.Background(), stage.ID)
	if err != buildErr {
		t.Errorf("expected build error, got %v", err)
	}
}

// TestDetails_ConvertConfigurationError is the primary regression test for
// CI-21565: when configuration conversion fails, the stage must be marked
// ERROR so the concurrency queue is unblocked.
func TestDetails_ConvertConfigurationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	build := testBuild()
	repo := testRepo()
	user := testUser()
	cfg := testConfig()
	convertErr := errors.New("Resource not accessible by personal access token")

	stages := mock.NewMockStageStore(ctrl)
	builds := mock.NewMockBuildStore(ctrl)
	repos := mock.NewMockRepositoryStore(ctrl)
	users := mock.NewMockUserStore(ctrl)
	configSvc := mock.NewMockConfigService(ctrl)
	converter := mock.NewMockConvertService(ctrl)
	events := mock.NewMockPubsub(ctrl)
	logs := mock.NewMockLogStream(ctrl)
	webhook := mock.NewMockWebhookSender(ctrl)
	statusSvc := mock.NewMockStatusService(ctrl)
	sched := mock.NewMockScheduler(ctrl)

	// Details() happy path up to Convert
	stages.EXPECT().Find(gomock.Any(), stage.ID).Return(stage, nil)
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)
	stages.EXPECT().List(gomock.Any(), stage.BuildID).Return([]*core.Stage{stage}, nil)
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	configSvc.EXPECT().Find(gomock.Any(), gomock.Any()).Return(cfg, nil)
	// Converter.Convert returns the PAT error — this is the reported bug trigger
	converter.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(nil, convertErr)

	// handleDetailsError Step 1: stage persisted as ERROR immediately
	stages.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if s.Status != core.StatusError {
				t.Errorf("want StatusError, got %s", s.Status)
			}
			if s.Error != convertErr.Error() {
				t.Errorf("want error %q, got %q", convertErr.Error(), s.Error)
			}
			return nil
		})

	// handleDetailsError Step 2: AfterAll teardown proceeds fully
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)
	stages.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	stages.EXPECT().ListSteps(gomock.Any(), build.ID).Return([]*core.Stage{stage}, nil)
	builds.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	events.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
	webhook.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	statusSvc.EXPECT().Send(gomock.Any(), user, gomock.Any()).Return(nil)

	m := &Manager{
		Builds:    builds,
		Config:    configSvc,
		Converter: converter,
		Events:    events,
		Logz:      logs,
		Repos:     repos,
		Scheduler: sched,
		Stages:    stages,
		Status:    statusSvc,
		Users:     users,
		Webhook:   webhook,
	}

	_, err := m.Details(context.Background(), stage.ID)
	if err != convertErr {
		t.Errorf("expected convert error, got %v", err)
	}
}

// TestDetails_ConfigFetchError verifies stage is marked ERROR when the
// .drone.yml fetch from GitHub/SCM fails.
func TestDetails_ConfigFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	build := testBuild()
	repo := testRepo()
	user := testUser()
	fetchErr := errors.New("failed to fetch config")

	stages := mock.NewMockStageStore(ctrl)
	builds := mock.NewMockBuildStore(ctrl)
	repos := mock.NewMockRepositoryStore(ctrl)
	users := mock.NewMockUserStore(ctrl)
	configSvc := mock.NewMockConfigService(ctrl)
	events := mock.NewMockPubsub(ctrl)
	logs := mock.NewMockLogStream(ctrl)
	webhook := mock.NewMockWebhookSender(ctrl)
	statusSvc := mock.NewMockStatusService(ctrl)
	sched := mock.NewMockScheduler(ctrl)

	stages.EXPECT().Find(gomock.Any(), stage.ID).Return(stage, nil)
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)
	stages.EXPECT().List(gomock.Any(), stage.BuildID).Return([]*core.Stage{stage}, nil)
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	configSvc.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil, fetchErr)

	stages.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if s.Status != core.StatusError {
				t.Errorf("want StatusError, got %s", s.Status)
			}
			return nil
		})

	// AfterAll teardown
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)
	stages.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	stages.EXPECT().ListSteps(gomock.Any(), build.ID).Return([]*core.Stage{stage}, nil)
	builds.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	events.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
	webhook.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	statusSvc.EXPECT().Send(gomock.Any(), user, gomock.Any()).Return(nil)

	m := &Manager{
		Builds:    builds,
		Config:    configSvc,
		Events:    events,
		Logz:      logs,
		Repos:     repos,
		Scheduler: sched,
		Stages:    stages,
		Status:    statusSvc,
		Users:     users,
		Webhook:   webhook,
	}

	_, err := m.Details(context.Background(), stage.ID)
	if err != fetchErr {
		t.Errorf("expected fetch error, got %v", err)
	}
}

// TestDetails_SecretsListError verifies stage is marked ERROR when
// listing repo secrets fails.
func TestDetails_SecretsListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stage := testStage()
	build := testBuild()
	repo := testRepo()
	user := testUser()
	cfg := testConfig()
	secretErr := errors.New("db: secrets unavailable")

	stages := mock.NewMockStageStore(ctrl)
	builds := mock.NewMockBuildStore(ctrl)
	repos := mock.NewMockRepositoryStore(ctrl)
	users := mock.NewMockUserStore(ctrl)
	configSvc := mock.NewMockConfigService(ctrl)
	converter := mock.NewMockConvertService(ctrl)
	secretStore := mock.NewMockSecretStore(ctrl)
	events := mock.NewMockPubsub(ctrl)
	logs := mock.NewMockLogStream(ctrl)
	webhook := mock.NewMockWebhookSender(ctrl)
	statusSvc := mock.NewMockStatusService(ctrl)
	sched := mock.NewMockScheduler(ctrl)

	stages.EXPECT().Find(gomock.Any(), stage.ID).Return(stage, nil)
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)
	stages.EXPECT().List(gomock.Any(), stage.BuildID).Return([]*core.Stage{stage}, nil)
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	configSvc.EXPECT().Find(gomock.Any(), gomock.Any()).Return(cfg, nil)
	converter.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(cfg, nil)
	secretStore.EXPECT().List(gomock.Any(), repo.ID).Return(nil, secretErr)

	stages.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s *core.Stage) error {
			if s.Status != core.StatusError {
				t.Errorf("want StatusError, got %s", s.Status)
			}
			return nil
		})

	// AfterAll teardown
	builds.EXPECT().Find(gomock.Any(), stage.BuildID).Return(build, nil)
	repos.EXPECT().Find(gomock.Any(), build.RepoID).Return(repo, nil)
	stages.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	stages.EXPECT().ListSteps(gomock.Any(), build.ID).Return([]*core.Stage{stage}, nil)
	builds.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	events.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
	webhook.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	users.EXPECT().Find(gomock.Any(), repo.UserID).Return(user, nil)
	statusSvc.EXPECT().Send(gomock.Any(), user, gomock.Any()).Return(nil)

	m := &Manager{
		Builds:    builds,
		Config:    configSvc,
		Converter: converter,
		Events:    events,
		Logz:      logs,
		Repos:     repos,
		Scheduler: sched,
		Secrets:   secretStore,
		Stages:    stages,
		Status:    statusSvc,
		Users:     users,
		Webhook:   webhook,
	}

	_, err := m.Details(context.Background(), stage.ID)
	if err != secretErr {
		t.Errorf("expected secrets error, got %v", err)
	}
}
