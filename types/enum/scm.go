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

// ScmType defines the different SCM types supported for CI.
type ScmType string

func (ScmType) Enum() []any { return toInterfaceSlice(scmTypes) }

var scmTypes = ([]ScmType{
	ScmTypeGitness,
	ScmTypeGithub,
	ScmTypeGitlab,
	ScmTypeUnknown,
})

const (
	ScmTypeUnknown ScmType = "UNKNOWN"
	ScmTypeGitness ScmType = "GITNESS"
	ScmTypeGithub  ScmType = "GITHUB"
	ScmTypeGitlab  ScmType = "GITLAB"
)

func AllSCMTypeStrings() []string {
	result := make([]string, len(scmTypes))
	for i, t := range scmTypes {
		result[i] = string(t)
	}
	return result
}
