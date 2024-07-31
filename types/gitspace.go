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

import (
	"github.com/harness/gitness/types/enum"
)

type GitspaceConfig struct {
	ID                              int64                     `json:"-"`
	Identifier                      string                    `json:"identifier"`
	Name                            string                    `json:"name"`
	IDE                             enum.IDEType              `json:"ide"`
	State                           enum.GitspaceStateType    `json:"state"`
	InfraProviderResourceID         int64                     `json:"-"`
	InfraProviderResourceIdentifier string                    `json:"resource_identifier"`
	CodeRepoURL                     string                    `json:"code_repo_url"`
	CodeRepoRef                     *string                   `json:"code_repo_ref"`
	CodeRepoType                    enum.GitspaceCodeRepoType `json:"code_repo_type"`
	Branch                          string                    `json:"branch"`
	DevcontainerPath                *string                   `json:"devcontainer_path,omitempty"`
	UserID                          string                    `json:"user_id"`
	SpaceID                         int64                     `json:"-"`
	CodeAuthType                    string                    `json:"-"`
	CodeAuthID                      string                    `json:"-"`
	IsDeleted                       bool                      `json:"-"`
	CodeRepoIsPrivate               bool                      `json:"-"`
	GitspaceInstance                *GitspaceInstance         `json:"instance"`
	SpacePath                       string                    `json:"space_path"`
	Created                         int64                     `json:"created"`
	Updated                         int64                     `json:"updated"`
}

type GitspaceInstance struct {
	ID               int64                          `json:"-"`
	GitSpaceConfigID int64                          `json:"-"`
	Identifier       string                         `json:"identifier"`
	URL              *string                        `json:"url,omitempty"`
	State            enum.GitspaceInstanceStateType `json:"state"`
	UserID           string                         `json:"-"`
	ResourceUsage    *string                        `json:"resource_usage"`
	LastUsed         int64                          `json:"last_used,omitempty"`
	TotalTimeUsed    int64                          `json:"total_time_used"`
	TrackedChanges   *string                        `json:"tracked_changes"`
	AccessKey        *string                        `json:"access_key,omitempty"`
	AccessType       enum.GitspaceAccessType        `json:"access_type"`
	MachineUser      *string                        `json:"machine_user,omitempty"`
	SpacePath        string                         `json:"space_path"`
	SpaceID          int64                          `json:"-"`
	Created          int64                          `json:"created"`
	Updated          int64                          `json:"updated"`
}

type GitspaceFilter struct {
	QueryFilter ListQueryFilter
	UserID      string
	SpaceIDs    []int64
}
