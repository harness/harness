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
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// State represents state of a background job.
type State string

// State enumeration.
const (
	JobStateScheduled State = "scheduled"
	JobStateRunning   State = "running"
	JobStateFinished  State = "finished"
	JobStateFailed    State = "failed"
	JobStateCanceled  State = "canceled"
)

var jobStates = sortEnum([]State{
	JobStateScheduled,
	JobStateRunning,
	JobStateFinished,
	JobStateFailed,
	JobStateCanceled,
})

func (State) Enum() []interface{} { return toInterfaceSlice(jobStates) }

func (s State) Sanitize() (State, bool) {
	return Sanitize(s, GetAllJobStates)
}

func GetAllJobStates() ([]State, State) {
	return jobStates, ""
}

// Priority represents priority of a background job.
type Priority int

// JobPriority enumeration.
const (
	JobPriorityNormal   Priority = 0
	JobPriorityElevated Priority = 1
)

func (s State) IsCompleted() bool {
	return s == JobStateFinished || s == JobStateFailed || s == JobStateCanceled
}

func sortEnum[T constraints.Ordered](slice []T) []T {
	slices.Sort(slice)
	return slice
}

func toInterfaceSlice[T interface{}](vals []T) []interface{} {
	res := make([]interface{}, len(vals))
	for i := range vals {
		res[i] = vals[i]
	}
	return res
}

func Sanitize[E constraints.Ordered](element E, all func() ([]E, E)) (E, bool) {
	allValues, defValue := all()
	var empty E
	if element == empty && defValue != empty {
		return defValue, true
	}
	idx, exists := slices.BinarySearch(allValues, element)
	if exists {
		return allValues[idx], true
	}
	return defValue, false
}
