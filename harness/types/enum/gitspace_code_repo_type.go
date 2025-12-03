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
	"encoding/json"
	"fmt"
)

type GitspaceCodeRepoType string

func (p GitspaceCodeRepoType) Enum() []interface{} {
	return toInterfaceSlice(codeRepoTypes)
}

var codeRepoTypes = []GitspaceCodeRepoType{
	CodeRepoTypeGithub,
	CodeRepoTypeGitlab,
	CodeRepoTypeHarnessCode,
	CodeRepoTypeBitbucket,
	CodeRepoTypeUnknown,
	CodeRepoTypeGitness,
	CodeRepoTypeGitlabOnPrem,
	CodeRepoTypeBitbucketServer,
	CodeRepoTypeGithubEnterprise,
}

const (
	CodeRepoTypeGithub           GitspaceCodeRepoType = "github"
	CodeRepoTypeGitlab           GitspaceCodeRepoType = "gitlab"
	CodeRepoTypeGitness          GitspaceCodeRepoType = "gitness"
	CodeRepoTypeHarnessCode      GitspaceCodeRepoType = "harness_code"
	CodeRepoTypeBitbucket        GitspaceCodeRepoType = "bitbucket"
	CodeRepoTypeUnknown          GitspaceCodeRepoType = "unknown"
	CodeRepoTypeGitlabOnPrem     GitspaceCodeRepoType = "gitlab_on_prem"
	CodeRepoTypeBitbucketServer  GitspaceCodeRepoType = "bitbucket_server"
	CodeRepoTypeGithubEnterprise GitspaceCodeRepoType = "github_enterprise"
)

func (p *GitspaceCodeRepoType) IsOnPrem() bool {
	if p == nil {
		return false
	}
	return *p == CodeRepoTypeGitlabOnPrem || *p == CodeRepoTypeBitbucketServer || *p == CodeRepoTypeGithubEnterprise
}

func (p *GitspaceCodeRepoType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*p = ""
		return nil
	}
	for _, v := range codeRepoTypes {
		if GitspaceCodeRepoType(s) == v {
			*p = v
			return nil
		}
	}
	return fmt.Errorf("invalid GitspaceCodeRepoType: %s", s)
}
