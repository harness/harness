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

type RepoActivityType string

const (
	RepoActivityTypeBranchCreated RepoActivityType = "branch-created"
	RepoActivityTypeBranchUpdated RepoActivityType = "branch-updated"
	RepoActivityTypeBranchDeleted RepoActivityType = "branch-deleted"
)

func (RepoActivityType) Enum() []any {
	return toInterfaceSlice(repoActivityTypes)
}

func (activityType RepoActivityType) Sanitize() (RepoActivityType, bool) {
	return Sanitize(activityType, GetAllRepoActivityTypes)
}

func GetAllRepoActivityTypes() ([]RepoActivityType, RepoActivityType) {
	return repoActivityTypes, ""
}

func ParseRepoActivityType(s string) (RepoActivityType, error) {
	activityType, ok := RepoActivityType(s).Sanitize()
	if !ok {
		return "", fmt.Errorf("unknown repo activity type provided: %q", s)
	}

	return activityType, nil
}

var repoActivityTypes = sortEnum([]RepoActivityType{
	RepoActivityTypeBranchCreated,
	RepoActivityTypeBranchUpdated,
	RepoActivityTypeBranchDeleted,
})
