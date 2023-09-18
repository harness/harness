// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// JobState represents state of a background job.
type JobState string

// JobState enumeration.
const (
	JobStateScheduled JobState = "scheduled"
	JobStateRunning   JobState = "running"
	JobStateFinished  JobState = "finished"
	JobStateFailed    JobState = "failed"
	JobStateCanceled  JobState = "canceled"
)

// JobPriority represents priority of a background job.
type JobPriority int

// JobPriority enumeration.
const (
	JobPriorityNormal   JobPriority = 0
	JobPriorityElevated JobPriority = 1
)

func GetCompletedJobState() []JobState {
	return []JobState{JobStateFinished, JobStateCanceled, JobStateFinished}
}
