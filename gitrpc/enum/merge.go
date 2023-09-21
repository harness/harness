// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package enum

import "github.com/harness/gitness/gitrpc/rpc"

// MergeMethod represents the approach to merge commits into base branch.
type MergeMethod string

const (
	// MergeMethodMerge create merge commit.
	MergeMethodMerge MergeMethod = "merge"
	// MergeMethodSquash squash commits into single commit before merging.
	MergeMethodSquash MergeMethod = "squash"
	// MergeMethodRebase rebase before merging.
	MergeMethodRebase MergeMethod = "rebase"
)

var MergeMethods = []MergeMethod{
	MergeMethodMerge,
	MergeMethodSquash,
	MergeMethodRebase,
}

func MergeMethodFromRPC(t rpc.MergeRequest_MergeMethod) MergeMethod {
	switch t {
	case rpc.MergeRequest_merge:
		return MergeMethodMerge
	case rpc.MergeRequest_squash:
		return MergeMethodSquash
	case rpc.MergeRequest_rebase:
		return MergeMethodRebase
	default:
		return MergeMethodMerge
	}
}

func (m MergeMethod) ToRPC() rpc.MergeRequest_MergeMethod {
	switch m {
	case MergeMethodMerge:
		return rpc.MergeRequest_merge
	case MergeMethodSquash:
		return rpc.MergeRequest_squash
	case MergeMethodRebase:
		return rpc.MergeRequest_rebase
	default:
		return rpc.MergeRequest_merge
	}
}

func (m MergeMethod) Sanitize() (MergeMethod, bool) {
	switch m {
	case MergeMethodMerge, MergeMethodSquash, MergeMethodRebase:
		return m, true
	default:
		return MergeMethodMerge, false
	}
}
