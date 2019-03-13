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

package manager

import (
	"context"
	"encoding/json"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

type setup struct {
	Builds core.BuildStore
	Events core.Pubsub
	Repos  core.RepositoryStore
	Steps  core.StepStore
	Stages core.StageStore
	Status core.StatusService
	Users  core.UserStore
}

func (s *setup) do(ctx context.Context, stage *core.Stage) error {
	logger := logrus.WithField("stage.id", stage.ID)

	build, err := s.Builds.Find(noContext, stage.BuildID)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot find the build")
		return err
	}

	repo, err := s.Repos.Find(noContext, build.RepoID)
	if err != nil {
		logger.WithError(err).WithFields(
			logrus.Fields{
				"build.number": build.Number,
				"build.id":     build.ID,
				"stage.id":     stage.ID,
				"repo.id":      build.RepoID,
			},
		).Warnln("manager: cannot find the repository")
		return err
	}

	logger = logger.WithFields(
		logrus.Fields{
			"build.number": build.Number,
			"build.id":     build.ID,
			"stage.id":     stage.ID,
			"repo.id":      build.RepoID,
		},
	)

	// // note that if multiple stages run concurrently it will attempt
	// // to create the watcher multiple times. The watcher is responsible
	// // for handling multiple concurrent requests and preventing duplication.
	// err = s.Watcher.Register(noContext, build.ID)
	// if err != nil {
	// 	logger.WithError(err).Warnln("manager: cannot create the watcher")
	// 	return err
	// }

	stage.Updated = time.Now().Unix()
	err = s.Stages.Update(noContext, stage)
	if err != nil {
		logger.WithError(err).
			WithField("stage.status", stage.Status).
			Warnln("manager: cannot update the stage")
		return err
	}

	for _, step := range stage.Steps {
		err := s.Steps.Create(noContext, step)
		if err != nil {
			logger.WithError(err).
				WithField("stage.status", stage.Status).
				WithField("step.name", step.Name).
				WithField("step.id", step.ID).
				Warnln("manager: cannot persist the step")
			return err
		}
	}

	updated, err := s.updateBuild(ctx, build)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot bupdate the build")
		return err
	}

	stages, err := s.Stages.ListSteps(noContext, build.ID)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot query build stages")
		return err
	}

	repo.Build = build
	repo.Build.Stages = stages
	data, _ := json.Marshal(repo)
	err = s.Events.Publish(noContext, &core.Message{
		Repository: repo.Slug,
		Visibility: repo.Visibility,
		Data:       data,
	})
	if err != nil {
		logger.Warnln("manager: cannot publish build event")
	}

	if updated {
		user, err := s.Users.Find(noContext, repo.UserID)
		if err != nil {
			logger.WithError(err).
				Warnln("manager: cannot find repository owner")
			return err
		}

		req := &core.StatusInput{
			Repo:  repo,
			Build: build,
		}
		err = s.Status.Send(noContext, user, req)
		if err != nil {
			logger.WithError(err).
				Warnln("manager: cannot publish status")
		}
	}

	return nil
}

// TODO(bradrydzewski) this should really be encapsulated into a single
// function call that internally uses a database transaction so that we
// can rollback if any operations fail.
func (s *setup) createSteps(ctx context.Context, stage *core.Stage) error {
	var errs error
	for _, step := range stage.Steps {
		err := s.Steps.Create(ctx, step)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// helper function that updates the build status from pending to running.
// This accounts for the fact that another agent may have already updated
// the build status, which may happen if two stages execute concurrently.
func (s *setup) updateBuild(ctx context.Context, build *core.Build) (bool, error) {
	if build.Status != core.StatusPending {
		return false, nil
	}
	build.Started = time.Now().Unix()
	build.Updated = time.Now().Unix()
	build.Status = core.StatusRunning
	err := s.Builds.Update(noContext, build)
	if err == db.ErrOptimisticLock {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
