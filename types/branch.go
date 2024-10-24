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

import "github.com/harness/gitness/git/sha"

type Branch struct {
	Name   string  `json:"name"`
	SHA    sha.SHA `json:"sha"`
	Commit *Commit `json:"commit,omitempty"`
}

type BranchExtended struct {
	Branch
	IsDefault        bool               `json:"is_default"`
	CheckSummary     *CheckCountSummary `json:"check_summary,omitempty"`
	Rules            []RuleInfo         `json:"rules,omitempty"`
	PullRequests     []*PullReq         `json:"pull_requests,omitempty"`
	CommitDivergence *CommitDivergence  `json:"commit_divergence,omitempty"`
}

type CreateBranchOutput struct {
	Branch
	DryRunRulesOutput
}

type DeleteBranchOutput struct {
	DryRunRulesOutput
}

// CommitDivergence contains the information of the count of converging commits between two refs.
type CommitDivergence struct {
	// Ahead is the count of commits the 'From' ref is ahead of the 'To' ref.
	Ahead int32 `json:"ahead"`
	// Behind is the count of commits the 'From' ref is behind the 'To' ref.
	Behind int32 `json:"behind"`
}
