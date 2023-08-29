// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	jobUIDOverdue  = "gitness:jobs:overdue"
	jobTypeOverdue = "gitness:jobs:overdue"
	jobCronOverdue = "*/20 * * * *" // every 20 min
)

type jobOverdue struct {
	store     store.JobStore
	mxManager lock.MutexManager
	scheduler *Scheduler
}

func newJobOverdue(jobStore store.JobStore, mxManager lock.MutexManager, scheduler *Scheduler) *jobOverdue {
	return &jobOverdue{
		store:     jobStore,
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

		if job.State == enum.JobStateScheduled {
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
