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

type GitspaceInstanceStateType string

func (GitspaceInstanceStateType) Enum() []interface{} {
	return toInterfaceSlice(gitspaceInstanceStateTypes)
}

var gitspaceInstanceStateTypes = []GitspaceInstanceStateType{
	GitspaceInstanceStateRunning,
	GitspaceInstanceStateUninitialized,
	GitspaceInstanceStateUnknown,
	GitspaceInstanceStateError,
	GitspaceInstanceStateDeleted,
	GitspaceInstanceStateStarting,
	GitspaceInstanceStateStopping,
	GitSpaceInstanceStateCleaning,
	GitspaceInstanceStateCleaned,
	GitSpaceInstanceStateResetting,
	GitspaceInstanceStatePendingCleanup,
}

const (
	GitspaceInstanceStateRunning        GitspaceInstanceStateType = "running"
	GitspaceInstanceStateUninitialized  GitspaceInstanceStateType = "uninitialized"
	GitspaceInstanceStateUnknown        GitspaceInstanceStateType = "unknown"
	GitspaceInstanceStateError          GitspaceInstanceStateType = "error"
	GitspaceInstanceStateStopped        GitspaceInstanceStateType = "stopped"
	GitspaceInstanceStateDeleted        GitspaceInstanceStateType = "deleted"
	GitspaceInstanceStateCleaned        GitspaceInstanceStateType = "cleaned"
	GitspaceInstanceStatePendingCleanup GitspaceInstanceStateType = "pending_cleanup"

	GitspaceInstanceStateStarting  GitspaceInstanceStateType = "starting"
	GitspaceInstanceStateStopping  GitspaceInstanceStateType = "stopping"
	GitSpaceInstanceStateCleaning  GitspaceInstanceStateType = "cleaning"
	GitSpaceInstanceStateResetting GitspaceInstanceStateType = "resetting"
)

func (g GitspaceInstanceStateType) IsFinalStatus() bool {
	//nolint:exhaustive
	switch g {
	case GitspaceInstanceStateDeleted,
		GitspaceInstanceStateError,
		GitspaceInstanceStateCleaned:
		return true
	default:
		return false
	}
}

func (g GitspaceInstanceStateType) IsBusyStatus() bool {
	//nolint:exhaustive
	switch g {
	case GitspaceInstanceStateStarting,
		GitspaceInstanceStateStopping,
		GitSpaceInstanceStateCleaning,
		GitSpaceInstanceStateResetting:
		return true
	default:
		return false
	}
}
