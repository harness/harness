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

type AITaskState string

func (AITaskState) Enum() []interface{} {
	return toInterfaceSlice(aiTaskStateTypes)
}

var aiTaskStateTypes = []AITaskState{
	AITaskStateUninitialized,
	AITaskStateRunning,
	AITaskStateCompleted,
	AITaskStateError,
}

const (
	AITaskStateUninitialized AITaskState = "uninitialized"
	AITaskStateRunning       AITaskState = "running"
	AITaskStateCompleted     AITaskState = "completed"
	AITaskStateError         AITaskState = "error"
)

func (a AITaskState) IsFinalStatus() bool {
	//nolint:exhaustive
	switch a {
	case AITaskStateCompleted,
		AITaskStateError:
		return true
	default:
		return false
	}
}

func (a AITaskState) IsActiveStatus() bool {
	//nolint:exhaustive
	switch a {
	case AITaskStateRunning:
		return true
	default:
		return false
	}
}
