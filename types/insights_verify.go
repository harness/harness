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

package types

// InsightsReleaseInfo describes a release seen in the events of an Insights project.
type InsightsReleaseInfo struct {
	Release string `json:"release"`
	// FirstSeen is the timestamp (epoch millis) of the first stored event of the release.
	FirstSeen int64 `json:"first_seen"`
}

// TaskLinkedRef is a task in a given state together with the reference of one of its links.
type TaskLinkedRef struct {
	TaskID int64  `json:"task_id"`
	RefID  string `json:"ref_id"`
	// Resolved is the time the task was resolved (epoch millis), zero if unknown.
	Resolved int64 `json:"resolved"`
}
