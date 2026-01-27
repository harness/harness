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

package audit

import (
	"time"

	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
)

// RepositoryObject is the object used for emitting repository related audits.
// TODO: ensure audit only takes audit related objects?
type RepositoryObject struct {
	types.Repository
	IsPublic bool `yaml:"is_public"`
}

type RegistryObject struct {
	registrytypes.Registry
}

type PullRequestObject struct {
	PullReq        types.PullReq
	RepoPath       string                 `yaml:"repo_path"`
	RuleViolations []types.RuleViolations `yaml:"rule_violations"`
	BypassMessage  string                 `yaml:"bypass_message,omitempty"`
}

type CommitObject struct {
	CommitSHA      string                 `yaml:"commit_sha"`
	RepoPath       string                 `yaml:"repo_path"`
	RuleViolations []types.RuleViolations `yaml:"rule_violations"`
}

type CommitTagObject struct {
	TagName        string                 `yaml:"tag_name"`
	RepoPath       string                 `yaml:"repo_path"`
	RuleViolations []types.RuleViolations `yaml:"rule_violations"`
}

type BranchObject struct {
	BranchName     string                 `yaml:"branch_name"`
	RepoPath       string                 `yaml:"repo_path"`
	RuleViolations []types.RuleViolations `yaml:"rule_violations"`
}

type RegistryUpstreamProxyConfigObject struct {
	ID         int64
	RegistryID int64
	Source     string
	URL        string
	AuthType   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CreatedBy  int64
	UpdatedBy  int64
}
