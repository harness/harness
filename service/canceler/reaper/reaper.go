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

package reaper

import (
	"context"
	"time"

	"github.com/drone/drone/core"
)

// Reaper finds and kills zombie jobs that are permanently
// stuck in a pending or running state.
type Reaper struct {
	Repos    core.RepositoryStore
	Builds   core.BuildStore
	Stages   core.StageStore
	Canceler core.Canceler
}

// TODO use multierror to aggregate errors encountered
// TODO use trace logging

func (r *Reaper) reap(ctx context.Context) error {
	ttl := time.Hour*24

	pending, err := r.Builds.Pending(ctx)
	if err != nil {
		return err
	}
	for _, build := range pending {
		// if a build is pending for longer than the maximum
		// pending time limit, the build is maybe cancelled.
		if isExceeded(build.Created, ttl, buffer) {
			err = r.reapMaybe(ctx, build)
			if err != nil {
				return err
			}
		}
	}

	running, err := r.Builds.Running(ctx)
	if err != nil {
		return err
	}
	for _, build := range running {
		// if a build is running for longer than the maximum
		// running time limit, the build is maybe cancelled.
		if isExceeded(build.Started, ttl, buffer) {
			err = r.reapMaybe(ctx, build)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reaper) reapMaybe(ctx context.Context, build *core.Build) error {
	repo, err := r.Repos.Find(ctx, build.RepoID)
	if err != nil {
		return err
	}

	// if the build status is pending we can immediately
	// cancel the build and all build stages.
	if build.Status == core.StatusPending {
		return r.Canceler.Cancel(ctx, repo, build)
	}

	stages, err := r.Stages.List(ctx, build.ID)
	if err != nil {
		return err
	}

	var started int64
	for _, stage := range stages {
		if stage.IsDone() {
			continue
		}
		if stage.Started > started  {
			started = stage.Started
		}
	}

	// if the build stages are all pending we can immediately
	// cancel the build.
	if started == 0 {
		return r.Canceler.Cancel(ctx, repo, build)
	}

	// if the build stage has exceeded the timeout by a reasonable
	// margin cancel the build and all build stages, else ignore.
	if isExceeded(started, time.Duration(repo.Timeout)*time.Minute, buffer) {
		return r.Canceler.Cancel(ctx, repo, build)
	}
	return nil
}
