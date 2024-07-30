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

package scm

import "github.com/harness/gitness/types"

type CodeRepositoryRequest struct {
	URL string `json:"url"`
}

type CodeRepositoryResponse struct {
	URL               string `json:"url"`
	Branch            string `json:"branch,omitempty"`
	CodeRepoIsPrivate bool   `json:"is_private"`
}

type (
	ResolvedDetails struct {
		*ResolvedCredentials
		DevcontainerConfig *types.DevcontainerConfig
	}

	// Credentials contains login and initialization information used
	// by an automated login process.
	Credentials struct {
		Email    string
		Name     string
		Password string
	}

	ResolvedCredentials struct {
		Branch      string
		CloneURL    string
		Credentials *Credentials
		RepoName    string
	}
)
