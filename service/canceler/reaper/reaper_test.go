// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package reaper

import (
	"context"
	"testing"
	"time"
	
	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
)

var nocontext = context.Background()

// this test confirms that the build is cancelled
// if the build status is pending.
func TestReapPendingMaybe(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockBuild := &core.Build{
		ID: 1,
		RepoID: 2,
		Status: core.StatusPending,
	}
	mockRepo := &core.Repository {
		ID: 2,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().Find(gomock.Any(), mockBuild.RepoID).Return(mockRepo, nil)


	canceler := mock.NewMockCanceler(controller)
	canceler.EXPECT().Cancel(gomock.Any(), mockRepo, mockBuild)
	
	r := &Reaper{
		Repos: repos,
		Stages: nil,
		Canceler: canceler,
	}

	r.reapMaybe(nocontext, mockBuild)
}

// this test confirms that the build is cancelled
// if the build status is running, and the stage
// started date is greater than the expiry date.
func TestReapRunningMaybe(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	defer func() {
		now = time.Now
	}()
	now = func() time.Time {
		return mustParse("2006-01-02T15:00:00")
	}

	mockBuild := &core.Build{
		ID: 1,
		RepoID: 2,
		Status: core.StatusRunning,
	}
	mockRepo := &core.Repository {
		ID: 2,
		Timeout: 60,
	}
	mockStages := []*core.Stage{
		{
			Status: core.StatusRunning,
			Started: mustParse("2006-01-02T13:00:00").Unix(), // running 2 hours, 1 hour longer than timeout
		},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().Find(gomock.Any(), mockBuild.RepoID).Return(mockRepo, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(mockStages, nil)

	canceler := mock.NewMockCanceler(controller)
	canceler.EXPECT().Cancel(gomock.Any(), mockRepo, mockBuild)
	
	r := &Reaper{
		Repos: repos,
		Stages: stages,
		Canceler: canceler,
	}

	r.reapMaybe(nocontext, mockBuild)
}

// this test confirms that if the build status is
// running, but all stages have a pending status,
// the build is cancelled (this likely points to some
// sort of race condition, and should not happen).
func TestReapRunningMaybe_AllStagesPending(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	defer func() {
		now = time.Now
	}()
	now = func() time.Time {
		return mustParse("2006-01-02T15:00:00")
	}

	mockBuild := &core.Build{
		ID: 1,
		RepoID: 2,
		Status: core.StatusRunning,
	}
	mockRepo := &core.Repository {
		ID: 2,
		Timeout: 60,
	}
	mockStages := []*core.Stage{
		{
			Status: core.StatusPending,
			Started: 0,
		},
		{
			Status: core.StatusPending,
			Started: 0,
		},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().Find(gomock.Any(), mockBuild.RepoID).Return(mockRepo, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(mockStages, nil)

	canceler := mock.NewMockCanceler(controller)
	canceler.EXPECT().Cancel(gomock.Any(), mockRepo, mockBuild)
	
	r := &Reaper{
		Repos: repos,
		Stages: stages,
		Canceler: canceler,
	}

	r.reapMaybe(nocontext, mockBuild)
}

// this test confirms that if the build status is
// running, but all stages have a finished status,
// the build is cancelled (this likely points to some
// sort of race condition, and should not happen).
func TestReapRunningMaybe_AllStagesFinished(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	defer func() {
		now = time.Now
	}()
	now = func() time.Time {
		return mustParse("2006-01-02T15:00:00")
	}

	mockBuild := &core.Build{
		ID: 1,
		RepoID: 2,
		Status: core.StatusRunning,
	}
	mockRepo := &core.Repository {
		ID: 2,
		Timeout: 60,
	}
	mockStages := []*core.Stage{
		{
			Status: core.StatusPassing,
			Started: mustParse("2006-01-02T14:40:00").Unix(),
		},
		{
			Status: core.StatusPassing,
			Started: mustParse("2006-01-02T14:50:00").Unix(),
		},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().Find(gomock.Any(), mockBuild.RepoID).Return(mockRepo, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(mockStages, nil)

	canceler := mock.NewMockCanceler(controller)
	canceler.EXPECT().Cancel(gomock.Any(), mockRepo, mockBuild)

	r := &Reaper{
		Repos: repos,
		Stages: stages,
		Canceler: canceler,
	}

	r.reapMaybe(nocontext, mockBuild)
}

// this test confirms that if the build status is
// running, but the stage start time has not exceeded
// the timeout period, the build is NOT cancelled.
func TestReapRunningMaybe_NotExpired(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	defer func() {
		now = time.Now
	}()
	now = func() time.Time {
		return mustParse("2006-01-02T15:00:00")
	}

	mockBuild := &core.Build{
		ID: 1,
		RepoID: 2,
		Status: core.StatusRunning,
	}
	mockRepo := &core.Repository {
		ID: 2,
		Timeout: 60,
	}
	mockStages := []*core.Stage{
		{
			Status: core.StatusPassing,
			Started: mustParse("2006-01-02T14:50:00").Unix(),
		},
		{
			Status: core.StatusRunning,
			Started: mustParse("2006-01-02T14:55:00").Unix(),
		},
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().Find(gomock.Any(), mockBuild.RepoID).Return(mockRepo, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().List(gomock.Any(), mockBuild.ID).Return(mockStages, nil)

	r := &Reaper{
		Repos: repos,
		Stages: stages,
		Canceler: nil,
	}

	r.reapMaybe(nocontext, mockBuild)
}

//
// Failure Scenarios
//

func TestReapRunningMaybe_ErrorGetRepo(t *testing.T) {

}

func TestReapRunningMaybe_ErrorListStages(t *testing.T) {

}
