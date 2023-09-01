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

	"github.com/rs/zerolog/log"
)

const (
	jobUIDPurge  = "gitness:jobs:purge"
	jobTypePurge = "gitness:jobs:purge"
	jobCronPurge = "15 */4 * * *" // every 4 hours at 15 minutes
)

type jobPurge struct {
	store     store.JobStore
	mxManager lock.MutexManager
	minOldAge time.Duration
}

func newJobPurge(jobStore store.JobStore, mxManager lock.MutexManager, minOldAge time.Duration) *jobPurge {
	if minOldAge < 0 {
		minOldAge = 0
	}

	return &jobPurge{
		store:     jobStore,
		mxManager: mxManager,
		minOldAge: minOldAge,
	}
}

func (j *jobPurge) Handle(ctx context.Context, _ string, _ ProgressReporter) (string, error) {
	mx, err := globalLock(ctx, j.mxManager)
	if err != nil {
		return "", fmt.Errorf("failed to obtain the lock to clean up old jobs")
	}

	defer func() {
		if err := mx.Unlock(ctx); err != nil {
			log.Err(err).Msg("failed to release global lock after cleaning up old jobs")
		}
	}()

	olderThan := time.Now().Add(-j.minOldAge)

	n, err := j.store.DeleteOld(ctx, olderThan)
	if err != nil {
		return "", fmt.Errorf("failed to purge old jobs")
	}

	result := "no old jobs found"
	if n > 0 {
		result = fmt.Sprintf("deleted %d old jobs", n)
	}

	return result, nil
}
