// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

type Job struct {
	UID                 string           `db:"job_uid"`
	Created             int64            `db:"job_created"`
	Updated             int64            `db:"job_updated"`
	Type                string           `db:"job_type"`
	Priority            enum.JobPriority `db:"job_priority"`
	Data                string           `db:"job_data"`
	Result              string           `db:"job_result"`
	MaxDurationSeconds  int              `db:"job_max_duration_seconds"`
	MaxRetries          int              `db:"job_max_retries"`
	State               enum.JobState    `db:"job_state"`
	Scheduled           int64            `db:"job_scheduled"`
	TotalExecutions     int              `db:"job_total_executions"`
	RunBy               string           `db:"job_run_by"`
	RunDeadline         int64            `db:"job_run_deadline"`
	RunProgress         int              `db:"job_run_progress"`
	LastExecuted        int64            `db:"job_last_executed"`
	IsRecurring         bool             `db:"job_is_recurring"`
	RecurringCron       string           `db:"job_recurring_cron"`
	ConsecutiveFailures int              `db:"job_consecutive_failures"`
	LastFailureError    string           `db:"job_last_failure_error"`
}

type JobStateChange struct {
	UID      string        `json:"uid"`
	State    enum.JobState `json:"state"`
	Progress int           `json:"progress"`
	Result   string        `json:"result"`
	Failure  string        `json:"failure"`
}

type JobProgress struct {
	State    enum.JobState `json:"state"`
	Progress int           `json:"progress"`
	Result   string        `json:"result,omitempty"`
	Failure  string        `json:"failure,omitempty"`
}
