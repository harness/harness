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

package job

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/lock"

	"github.com/rs/zerolog/log"
)

const (
	jobUIDOverdue  = "gitness:jobs:overdue"
	jobTypeOverdue = "gitness:jobs:overdue"
	jobCronOverdue = "*/20 * * * *" // every 20 min
)

type jobOverdue struct {
	store     Store
	mxManager lock.MutexManager
	scheduler *Scheduler
}

func newJobOverdue(store Store, mxManager lock.MutexManager, scheduler *Scheduler) *jobOverdue {
	return &jobOverdue{
		store:     store,
		mxManager: mxManager,
		scheduler: scheduler,
	}
}

// Handle reclaims overdue jobs. Normally this shouldn't happen.
// But, it can occur if DB update after a job execution fails,
// or the server suddenly terminates while the job is still running.
func (j *jobOverdue) Handle(ctx context.Context, _ string, _ ProgressReporter) (string, error) {
	mx, err := globalLock(ctx, j.mxManager)
	if err != nil {
		return "", fmt.Errorf("failed to obtain the lock to reclaim overdue jobs")
	}

	defer func() {
		if err := mx.Unlock(ctx); err != nil {
			log.Err(err).Msg("failed to release global lock after reclaiming overdue jobs")
		}
	}()

	overdueJobs, err := j.store.ListDeadlineExceeded(ctx, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to list overdue jobs")
	}

	if len(overdueJobs) == 0 {
		return "", nil
	}

	var minScheduled time.Time

	for _, job := range overdueJobs {
		const errorMessage = "deadline exceeded"
		postExec(job, "", errorMessage)

		err = j.store.UpdateExecution(ctx, job)
		if err != nil {
			return "", fmt.Errorf("failed update overdue job")
		}

		if job.State == JobStateScheduled {
			scheduled := time.UnixMilli(job.Scheduled)
			if minScheduled.IsZero() || minScheduled.After(scheduled) {
				minScheduled = scheduled
			}
		}
	}

	if !minScheduled.IsZero() {
		j.scheduler.scheduleProcessing(minScheduled)
	}

	result := fmt.Sprintf("found %d overdue jobs", len(overdueJobs))

	return result, nil
}
