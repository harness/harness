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

import "fmt"

type GitspaceStateType string

func (GitspaceStateType) Enum() []interface{} {
	return toInterfaceSlice(gitspaceStateTypes)
}

var gitspaceStateTypes = []GitspaceStateType{
	GitspaceStateRunning,
	GitspaceStateStopped,
	GitspaceStateError,
	GitspaceStateUninitialized,
	GitspaceStateStarting,
	GitspaceStateStopping,
}

const (
	GitspaceStateRunning       GitspaceStateType = "running"
	GitspaceStateStopped       GitspaceStateType = "stopped"
	GitspaceStateStarting      GitspaceStateType = "starting"
	GitspaceStateStopping      GitspaceStateType = "stopping"
	GitspaceStateError         GitspaceStateType = "error"
	GitspaceStateUninitialized GitspaceStateType = "uninitialized"
)

func GetGitspaceStateFromInstance(
	instanceState GitspaceInstanceStateType,
) (GitspaceStateType, error) {
	switch instanceState {
	case GitspaceInstanceStateRunning:
		return GitspaceStateRunning, nil
	case GitspaceInstanceStateDeleted:
		return GitspaceStateStopped, nil
	case GitspaceInstanceStateStarting:
		return GitspaceStateStarting, nil
	case GitspaceInstanceStateStopping:
		return GitspaceStateStopping, nil
	case GitspaceInstanceStateUninitialized:
		return GitspaceStateUninitialized, nil
	case GitspaceInstanceStateError,
		GitspaceInstanceStateUnknown:
		return GitspaceStateError, nil
	default:
		return GitspaceStateError, fmt.Errorf("unsupported gitspace instance state %s", string(instanceState))
	}
}
