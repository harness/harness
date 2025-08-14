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

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ job.Store = (*JobStore)(nil)

func NewJobStore(db *sqlx.DB) *JobStore {
	return &JobStore{
		db: db,
	}
}

type JobStore struct {
	db *sqlx.DB
}

const (
	jobColumns = `
		 job_uid
		,job_created
		,job_updated
		,job_type
		,job_priority
		,job_data
		,job_result
		,job_max_duration_seconds
		,job_max_retries
		,job_state
		,job_scheduled
		,job_total_executions
		,job_run_by
		,job_run_deadline
		,job_run_progress
		,job_last_executed
		,job_is_recurring
		,job_recurring_cron
		,job_consecutive_failures
		,job_last_failure_error
		,job_group_id`

	jobSelectBase = `
	SELECT` + jobColumns + `
	FROM jobs`
)

// Find fetches a job by its unique identifier.
func (s *JobStore) Find(ctx context.Context, uid string) (*job.Job, error) {
	const sqlQuery = jobSelectBase + `
	WHERE job_uid = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	result := &job.Job{}
	if err := db.GetContext(ctx, result, sqlQuery, uid); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find job by uid")
	}

	return result, nil
}

// DeleteByGroupID deletes all jobs for a group id.
func (s *JobStore) DeleteByGroupID(ctx context.Context, groupID string) (int64, error) {
	stmt := database.Builder.
		Delete("jobs").
		Where("(job_group_id = ?)", groupID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert delete by group id jobs query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to execute delete jobs by group id query")
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to get number of deleted jobs in group")
	}

	return n, nil
}

// ListByGroupID fetches all jobs for a group id.
func (s *JobStore) ListByGroupID(ctx context.Context, groupID string) ([]*job.Job, error) {
	const sqlQuery = jobSelectBase + `
	WHERE job_group_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := make([]*job.Job, 0)
	if err := db.SelectContext(ctx, &dst, sqlQuery, groupID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find job by group id")
	}

	return dst, nil
}

// Create creates a new job.
func (s *JobStore) Create(ctx context.Context, job *job.Job) error {
	const sqlQuery = `
		INSERT INTO jobs (` + jobColumns + `
		) VALUES (
			 :job_uid
			,:job_created
			,:job_updated
			,:job_type
			,:job_priority
			,:job_data
			,:job_result
			,:job_max_duration_seconds
			,:job_max_retries
			,:job_state
			,:job_scheduled
			,:job_total_executions
			,:job_run_by
			,:job_run_deadline
			,:job_run_progress
			,:job_last_executed
			,:job_is_recurring
			,:job_recurring_cron
			,:job_consecutive_failures
			,:job_last_failure_error
			,:job_group_id
		)`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, job)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind job object")
	}

	if _, err := db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// Upsert creates or updates a job. If the job didn't exist it will insert it in the database,
// otherwise it will update it but only if its definition has changed.
func (s *JobStore) Upsert(ctx context.Context, job *job.Job) error {
	const sqlQuery = `
		INSERT INTO jobs (` + jobColumns + `
		) VALUES (
			 :job_uid
			,:job_created
			,:job_updated
			,:job_type
			,:job_priority
			,:job_data
			,:job_result
			,:job_max_duration_seconds
			,:job_max_retries
			,:job_state
			,:job_scheduled
			,:job_total_executions
			,:job_run_by
			,:job_run_deadline
			,:job_run_progress
			,:job_last_executed
			,:job_is_recurring
			,:job_recurring_cron
			,:job_consecutive_failures
			,:job_last_failure_error
			,:job_group_id
		)
		ON CONFLICT (job_uid) DO
		UPDATE SET
			 job_updated = :job_updated
			,job_type = :job_type
			,job_priority = :job_priority
			,job_data = :job_data
			,job_result = :job_result
			,job_max_duration_seconds = :job_max_duration_seconds
			,job_max_retries = :job_max_retries
			,job_state = :job_state
			,job_scheduled = :job_scheduled
			,job_is_recurring = :job_is_recurring
			,job_recurring_cron = :job_recurring_cron
		WHERE
			jobs.job_type <> :job_type OR
			jobs.job_priority <> :job_priority OR
			jobs.job_data <> :job_data OR
			jobs.job_max_duration_seconds <> :job_max_duration_seconds OR
			jobs.job_max_retries <> :job_max_retries OR
			jobs.job_is_recurring <> :job_is_recurring OR
			jobs.job_recurring_cron <> :job_recurring_cron`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, job)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind job object")
	}

	if _, err := db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Upsert query failed")
	}

	return nil
}

// UpdateDefinition is used to update a job definition.
func (s *JobStore) UpdateDefinition(ctx context.Context, job *job.Job) error {
	const sqlQuery = `
	UPDATE jobs
	SET
		 job_updated = :job_updated
		,job_type = :job_type
		,job_priority = :job_priority
		,job_data = :job_data
		,job_result = :job_result
		,job_max_duration_seconds = :job_max_duration_seconds
		,job_max_retries = :job_max_retries
		,job_state = :job_state
		,job_scheduled = :job_scheduled
		,job_is_recurring = :job_is_recurring
		,job_recurring_cron = :job_recurring_cron
		,job_group_id = :job_group_id
	WHERE job_uid = :job_uid`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, job)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind job object for update")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update job definition")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrResourceNotFound
	}

	return nil
}

// UpdateExecution is used to update a job before and after execution.
func (s *JobStore) UpdateExecution(ctx context.Context, job *job.Job) error {
	const sqlQuery = `
	UPDATE jobs
	SET
		 job_updated = :job_updated
		,job_result = :job_result
		,job_state = :job_state
		,job_scheduled = :job_scheduled
		,job_total_executions = :job_total_executions
		,job_run_by = :job_run_by
		,job_run_deadline = :job_run_deadline
		,job_last_executed = :job_last_executed
		,job_consecutive_failures = :job_consecutive_failures
		,job_last_failure_error = :job_last_failure_error
	WHERE job_uid = :job_uid`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, job)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind job object for update")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update job execution")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrResourceNotFound
	}

	return nil
}

func (s *JobStore) UpdateProgress(ctx context.Context, job *job.Job) error {
	const sqlQuery = `
	UPDATE jobs
	SET
		 job_updated = :job_updated
		,job_result = :job_result
	    ,job_run_progress = :job_run_progress
	WHERE job_uid = :job_uid AND job_state = 'running'`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, job)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind job object for update")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update job progress")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrResourceNotFound
	}

	return nil
}

// CountRunning returns number of jobs that are currently being run.
func (s *JobStore) CountRunning(ctx context.Context) (int, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("jobs").
		Where("job_state = ?", enum.JobStateRunning)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert count running jobs query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed executing count running jobs query")
	}

	return int(count), nil
}

// ListReady returns a list of jobs that are ready for execution:
// The jobs with state="scheduled" and scheduled time in the past.
func (s *JobStore) ListReady(ctx context.Context, now time.Time, limit int) ([]*job.Job, error) {
	stmt := database.Builder.
		Select(jobColumns).
		From("jobs").
		Where("job_state = ?", enum.JobStateScheduled).
		Where("job_scheduled <= ?", now.UnixMilli()).
		OrderBy("job_priority desc, job_scheduled asc, job_uid asc").
		Limit(uint64(limit)) //nolint:gosec

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert list scheduled jobs query to sql: %w", err)
	}

	result := make([]*job.Job, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &result, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to execute list scheduled jobs query")
	}

	return result, nil
}

// ListDeadlineExceeded returns a list of jobs that have exceeded their execution deadline.
func (s *JobStore) ListDeadlineExceeded(ctx context.Context, now time.Time) ([]*job.Job, error) {
	stmt := database.Builder.
		Select(jobColumns).
		From("jobs").
		Where("job_state = ?", enum.JobStateRunning).
		Where("job_run_deadline < ?", now.UnixMilli()).
		OrderBy("job_run_deadline asc")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert list overdue jobs query to sql: %w", err)
	}

	result := make([]*job.Job, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &result, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to execute list overdue jobs query")
	}

	return result, nil
}

// NextScheduledTime returns a scheduled time of the next ready job or zero time if no such job exists.
func (s *JobStore) NextScheduledTime(ctx context.Context, now time.Time) (time.Time, error) {
	stmt := database.Builder.
		Select("job_scheduled").
		From("jobs").
		Where("job_state = ?", enum.JobStateScheduled).
		Where("job_scheduled > ?", now.UnixMilli()).
		OrderBy("job_scheduled asc").
		Limit(1)

	query, args, err := stmt.ToSql()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to convert next scheduled time query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var result int64

	err = db.QueryRowContext(ctx, query, args...).Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, database.ProcessSQLErrorf(ctx, err, "failed to execute next scheduled time query")
	}

	return time.UnixMilli(result), nil
}

// DeleteOld removes non-recurring jobs that have finished execution or have failed.
func (s *JobStore) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	stmt := database.Builder.
		Delete("jobs").
		Where("(job_state = ? OR job_state = ? OR job_state = ?)",
			enum.JobStateFinished, enum.JobStateFailed, enum.JobStateCanceled).
		Where("job_is_recurring = false").
		Where("job_last_executed < ?", olderThan.UnixMilli())

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert delete done jobs query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to execute delete done jobs query")
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to get number of deleted jobs")
	}

	return n, nil
}

// DeleteByID deletes a job by its unique identifier.
func (s *JobStore) DeleteByUID(ctx context.Context, jobUID string) error {
	stmt := database.Builder.
		Delete("jobs").
		Where("job_uid = ?", jobUID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert delete job query to sql: %w", err)
	}
	db := dbtx.GetAccessor(ctx, s.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to execute delete job query")
	}
	return nil
}
