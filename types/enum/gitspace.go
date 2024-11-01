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

import (
	"strings"
)

// GitspaceSort represents gitspace sort order.
type GitspaceSort string

// GitspaceSort enumeration.
const (
	GitspaceSortLastUsed GitspaceSort = lastUsed
	GitspaceSortCreated  GitspaceSort = created
)

var GitspaceSorts = sortEnum([]GitspaceSort{
	GitspaceSortLastUsed,
	GitspaceSortCreated,
})

func (GitspaceSort) Enum() []interface{} { return toInterfaceSlice(GitspaceSorts) }

// ParseGitspaceSort parses the gitspace sort attribute string
// and returns the equivalent enumeration.
func ParseGitspaceSort(s string) GitspaceSort {
	switch strings.ToLower(s) {
	case lastUsed:
		return GitspaceSortLastUsed
	case created, createdAt:
		return GitspaceSortCreated
	default:
		return GitspaceSortLastUsed
	}
}
