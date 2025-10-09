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
	jobUIDPurge  = "gitness:jobs:purge"
	jobTypePurge = "gitness:jobs:purge"
	jobCronPurge = "15 */4 * * *" // every 4 hours at 15 minutes
)

type jobPurge struct {
	store     Store
	mxManager lock.MutexManager
	minOldAge time.Duration
}

func newJobPurge(store Store, mxManager lock.MutexManager, minOldAge time.Duration) *jobPurge {
	if minOldAge < 0 {
		minOldAge = 0
	}

	return &jobPurge{
		store:     store,
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
