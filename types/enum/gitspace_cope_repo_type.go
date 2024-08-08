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

type GitspaceCodeRepoType string

func (GitspaceCodeRepoType) Enum() []interface{} { return toInterfaceSlice(codeRepoTypes) }

var codeRepoTypes = []GitspaceCodeRepoType{
	CodeRepoTypeGithub,
	CodeRepoTypeGitlab,
	CodeRepoTypeHarnessCode,
	CodeRepoTypeBitbucket,
	CodeRepoTypeUnknown,
	CodeRepoTypeGitness,
}

const (
	CodeRepoTypeGithub      GitspaceCodeRepoType = "github"
	CodeRepoTypeGitlab      GitspaceCodeRepoType = "gitlab"
	CodeRepoTypeGitness     GitspaceCodeRepoType = "gitness"
	CodeRepoTypeHarnessCode GitspaceCodeRepoType = "harness_code"
	CodeRepoTypeBitbucket   GitspaceCodeRepoType = "bitbucket"
	CodeRepoTypeUnknown     GitspaceCodeRepoType = "unknown"
)
