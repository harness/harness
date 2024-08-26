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

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CodeRepositoryRequest struct {
	URL            string
	RepoType       enum.GitspaceCodeRepoType
	UserIdentifier string
	UserID         int64
	SpacePath      string
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

	RepositoryFilter struct {
		SpaceID int64  `json:"space_id"`
		Page    int    `json:"page"`
		Size    int    `json:"size"`
		Query   string `json:"query"`
	}

	BranchFilter struct {
		SpaceID    int64  `json:"space_id"`
		Repository string `json:"repo"`
		Query      string `json:"query"`
		Page       int    `json:"page"`
		Size       int    `json:"size"`
	}

	Repository struct {
		Name          string `json:"name"`
		DefaultBranch string `json:"default_branch"`
		// git urls
		GitURL    string `json:"git_url"`
		GitSSHURL string `json:"git_ssh_url,omitempty"`
	}

	Branch struct {
		Name string `json:"name"`
		SHA  string `json:"sha"`
	}
)
