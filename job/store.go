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
	"time"
)

type Store interface {
	// Find fetches a job by its unique identifier.
	Find(ctx context.Context, uid string) (*Job, error)

	// ListByGroupID fetches all jobs for a group id
	ListByGroupID(ctx context.Context, groupID string) ([]*Job, error)

	// DeleteByGroupID deletes all jobs for a group id
	DeleteByGroupID(ctx context.Context, groupID string) (int64, error)

	// Create is used to create a new job.
	Create(ctx context.Context, job *Job) error

	// Upsert will insert the job in the database if the job didn't already exist,
	// or it will update the existing one but only if its definition has changed.
	Upsert(ctx context.Context, job *Job) error

	// UpdateDefinition is used to update a job definition.
	UpdateDefinition(ctx context.Context, job *Job) error

	// UpdateExecution is used to update a job before and after execution.
	UpdateExecution(ctx context.Context, job *Job) error

	// UpdateProgress is used to update a job progress data.
	UpdateProgress(ctx context.Context, job *Job) error

	// CountRunning returns number of jobs that are currently being run.
	CountRunning(ctx context.Context) (int, error)

	// ListReady returns a list of jobs that are ready for execution.
	ListReady(ctx context.Context, now time.Time, limit int) ([]*Job, error)

	// ListDeadlineExceeded returns a list of jobs that have exceeded their execution deadline.
	ListDeadlineExceeded(ctx context.Context, now time.Time) ([]*Job, error)

	// NextScheduledTime returns a scheduled time of the next ready job.
	NextScheduledTime(ctx context.Context, now time.Time) (time.Time, error)

	// DeleteOld removes non-recurring jobs that have finished execution or have failed.
	DeleteOld(ctx context.Context, olderThan time.Time) (int64, error)

	// DeleteByUID deletes a job by its unique identifier.
	DeleteByUID(ctx context.Context, jobUID string) error
}
