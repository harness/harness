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

var jobStates = sortEnum([]JobState{
	JobStateScheduled,
	JobStateRunning,
	JobStateFinished,
	JobStateFailed,
	JobStateCanceled,
})

func (JobState) Enum() []interface{} { return toInterfaceSlice(jobStates) }
func (s JobState) Sanitize() (JobState, bool) {
	return Sanitize(s, GetAllJobStates)
}
func GetAllJobStates() ([]JobState, JobState) {
	return jobStates, ""
}

// JobPriority represents priority of a background job.
type JobPriority int

// JobPriority enumeration.
const (
	JobPriorityNormal   JobPriority = 0
	JobPriorityElevated JobPriority = 1
)

func (s JobState) IsCompleted() bool {
	return s == JobStateFinished || s == JobStateFailed || s == JobStateCanceled
}
