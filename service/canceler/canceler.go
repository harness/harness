// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package canceler

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"time"

	"github.com/drone/drone/core"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

var noContext = context.Background()

type service struct {
	builds    core.BuildStore
	events    core.Pubsub
	repos     core.RepositoryStore
	scheduler core.Scheduler
	stages    core.StageStore
	status    core.StatusService
	steps     core.StepStore
	users     core.UserStore
	webhooks  core.WebhookSender
}

// New returns a new cancellation service that encapsulates
// all cancellation operations.
func New(
	builds core.BuildStore,
	events core.Pubsub,
	repos core.RepositoryStore,
	scheduler core.Scheduler,
	stages core.StageStore,
	status core.StatusService,
	steps core.StepStore,
	users core.UserStore,
	webhooks core.WebhookSender,
) core.Canceler {
	return &service{
		builds:    builds,
		events:    events,
		repos:     repos,
		scheduler: scheduler,
		stages:    stages,
		status:    status,
		steps:     steps,
		users:     users,
		webhooks:  webhooks,
	}
}

// Cancel cancels a build.
func (s *service) Cancel(ctx context.Context, repo *core.Repository, build *core.Build) error {
	return s.cancel(ctx, repo, build, core.StatusKilled)
}

// CancelPending cancels all pending builds of the same event
// and reference with lower build numbers.
func (s *service) CancelPending(ctx context.Context, repo *core.Repository, build *core.Build) error {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()

	// switch {
	// case repo.CancelPulls && build.Event == core.EventPullRequest:
	// case repo.CancelPush && build.Event == core.EventPush:
	// default:
	// 	return nil
	// }

	switch build.Event {
	// on the push and pull request builds can be automatically
	// cancelled by the system.
	case core.EventPush, core.EventPullRequest:
	default:
		return nil
	}

	// get a list of all incomplete builds from the database
	// for all repositories. this will need to be filtered.
	incomplete, err := s.repos.ListIncomplete(ctx)
	if err != nil {
		return err
	}

	var result error
	for _, item := range incomplete {
		// ignore incomplete items in the list that do
		// not match the repository or build
		if !match(build, item) {
			continue
		}

		err := s.cancel(ctx, repo, item.Build, core.StatusSkipped)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

func (s *service) cancel(ctx context.Context, repo *core.Repository, build *core.Build, status string) error {
	logger := logrus.WithFields(
		logrus.Fields{
			"repo":   repo.Slug,
			"ref":    build.Ref,
			"build":  build.Number,
			"event":  build.Event,
			"status": build.Status,
		},
	)

	// do not cancel the build if the build status is
	// complete. only cancel the build if the status is
	// running or pending.
	switch build.Status {
	case core.StatusPending, core.StatusRunning:
	default:
		return nil
	}

	// update the build status to killed. if the update fails
	// due to an optimistic lock error it means the build has
	// already started, and should now be ignored.
	build.Status = status
	build.Finished = time.Now().Unix()
	if build.Started == 0 {
		build.Started = time.Now().Unix()
	}

	err := s.builds.Update(ctx, build)
	if err != nil {
		logger.WithError(err).
			Warnln("canceler: cannot update build status to cancelled")
		return err
	}

	// notify the scheduler to cancel the build. this will
	// instruct runners subscribing to the scheduler to
	// cancel execution.
	err = s.scheduler.Cancel(ctx, build.ID)
	if err != nil {
		logger.WithError(err).
			Warnln("canceler: cannot signal cancelled build is complete")
	}

	// update the commit status in the remote source
	// control management system.
	user, err := s.users.Find(ctx, repo.UserID)
	if err == nil {
		err := s.status.Send(ctx, user, &core.StatusInput{
			Repo:  repo,
			Build: build,
		})
		if err != nil {
			logger.WithError(err).
				Debugln("canceler: cannot set status")
		}
	}

	stages, err := s.stages.ListSteps(ctx, build.ID)
	if err != nil {
		logger.WithError(err).
			Debugln("canceler: cannot list build stages")
	}

	// update the status of all steps to indicate they
	// were killed or skipped.
	for _, stage := range stages {
		if stage.IsDone() {
			continue
		}
		if stage.Started != 0 {
			stage.Status = core.StatusKilled
		} else {
			stage.Status = core.StatusSkipped
			stage.Started = time.Now().Unix()
		}
		stage.Stopped = time.Now().Unix()
		err := s.stages.Update(ctx, stage)
		if err != nil {
			logger.WithError(err).
				WithField("stage", stage.Number).
				Debugln("canceler: cannot update stage status")
		}

		// update the status of all steps to indicate they
		// were killed or skipped.
		for _, step := range stage.Steps {
			if step.IsDone() {
				continue
			}
			if step.Started != 0 {
				step.Status = core.StatusKilled
			} else {
				step.Status = core.StatusSkipped
				step.Started = time.Now().Unix()
			}
			step.Stopped = time.Now().Unix()
			step.ExitCode = 130
			err := s.steps.Update(ctx, step)
			if err != nil {
				logger.WithError(err).
					WithField("stage", stage.Number).
					WithField("step", step.Number).
					Debugln("canceler: cannot update step status")
			}
		}
	}

	logger.WithError(err).
		Debugln("canceler: successfully cancelled build")

	build.Stages = stages

	// trigger a pubsub event to notify subscribers that
	// the build was cancelled. Specifically, this should
	// live update the user interface.
	repoCopy := new(core.Repository)
	*repoCopy = *repo
	repoCopy.Build = build
	repoCopy.Build.Stages = stages
	data, _ := json.Marshal(repoCopy)
	err = s.events.Publish(noContext, &core.Message{
		Repository: repo.Slug,
		Visibility: repo.Visibility,
		Data:       data,
	})
	if err != nil {
		logger.WithError(err).
			Warnln("canceler: cannot publish cancel event")
	}

	// trigger a webhook to notify subscribing systems that
	// the build was cancelled.
	payload := &core.WebhookData{
		Event:  core.WebhookEventBuild,
		Action: core.WebhookActionUpdated,
		Repo:   repo,
		Build:  build,
	}
	err = s.webhooks.Send(ctx, payload)
	if err != nil {
		logger.WithError(err).
			Warnln("manager: cannot send global webhook")
	}

	return nil
}
