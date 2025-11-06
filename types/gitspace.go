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
	"fmt"

	"github.com/harness/gitness/types/enum"
)

const emptyGitspaceInstanceState = ""

type GitspaceConfig struct {
	ID                    int64                        `json:"-"`
	Identifier            string                       `json:"identifier"`
	Name                  string                       `json:"name"`
	IDE                   enum.IDEType                 `json:"ide"`
	State                 enum.GitspaceStateType       `json:"state"`
	SpaceID               int64                        `json:"-"`
	IsDeleted             bool                         `json:"-"`
	IsMarkedForDeletion   bool                         `json:"-"`
	IsMarkedForReset      bool                         `json:"is_marked_for_reset"`
	IsMarkedForInfraReset bool                         `json:"is_marked_for_infra_reset"`
	GitspaceInstance      *GitspaceInstance            `json:"instance"`
	SpacePath             string                       `json:"space_path"`
	Created               int64                        `json:"created"`
	Updated               int64                        `json:"updated"`
	SSHTokenIdentifier    string                       `json:"ssh_token_identifier"`
	InfraProviderResource InfraProviderResource        `json:"resource"`
	LogKey                string                       `json:"log_key"`
	InitializeLogKey      string                       `json:"initialize_log_key"`
	AIAgents              []enum.AIAgent               `json:"ai_agents,omitempty"`
	AIAuth                map[enum.AIAgent]AIAgentAuth `json:"-"`
	CodeRepo
	GitspaceUser
	Connectors []PlatformConnector `json:"-"`
}

type CodeRepo struct {
	URL              string                    `json:"code_repo_url"`
	Ref              *string                   `json:"code_repo_ref"`
	Type             enum.GitspaceCodeRepoType `json:"code_repo_type"`
	Branch           string                    `json:"branch"`
	BranchURL        string                    `json:"branch_url,omitempty"`
	DevcontainerPath *string                   `json:"devcontainer_path,omitempty"`
	IsPrivate        bool                      `json:"code_repo_is_private"`
	AuthType         string                    `json:"-"`
	AuthID           string                    `json:"-"`
}

type GitspaceUser struct {
	ID          *int64 `json:"-"`
	Identifier  string `json:"user_id"`
	Email       string `json:"user_email"`
	DisplayName string `json:"user_display_name"`
}

type GitspaceInstance struct {
	ID                int64                          `json:"-"`
	GitSpaceConfigID  int64                          `json:"-"`
	Identifier        string                         `json:"identifier"`
	URL               *string                        `json:"url,omitempty"`
	SSHCommand        *string                        `json:"ssh_command,omitempty"`
	PluginURL         *string                        `json:"plugin_url,omitempty"`
	State             enum.GitspaceInstanceStateType `json:"state"`
	UserID            string                         `json:"-"`
	ResourceUsage     *string                        `json:"resource_usage"`
	LastUsed          *int64                         `json:"last_used,omitempty"`
	TotalTimeUsed     int64                          `json:"total_time_used"`
	AccessKey         *string                        `json:"access_key,omitempty"`
	AccessType        enum.GitspaceAccessType        `json:"access_type"`
	AccessKeyRef      *string                        `json:"access_key_ref"`
	MachineUser       *string                        `json:"machine_user,omitempty"`
	SpacePath         string                         `json:"space_path"`
	SpaceID           int64                          `json:"-"`
	Created           int64                          `json:"created"`
	Updated           int64                          `json:"updated"`
	LastHeartbeat     *int64                         `json:"last_heartbeat,omitempty"`
	ActiveTimeStarted *int64                         `json:"active_time_started,omitempty"`
	ActiveTimeEnded   *int64                         `json:"active_time_ended,omitempty"`
	HasGitChanges     *bool                          `json:"has_git_changes,omitempty"`
	ErrorMessage      *string                        `json:"error_message,omitempty"`
}

type GitspaceFilter struct {
	QueryFilter          ListQueryFilter
	Sort                 enum.GitspaceSort `json:"sort"`
	Order                enum.Order        `json:"order"`
	Owner                enum.GitspaceOwner
	GitspaceFilterStates []enum.GitspaceFilterState
	CodeRepoTypes        []enum.GitspaceCodeRepoType
	Deleted              *bool // not nil when we want to add this filter
	MarkedForDeletion    *bool // not nil when we want to add this filter
	GitspaceInstanceFilter
	ScopeFilter
}

type ScopeFilter struct {
	// each org will be of the format "orgID"
	Orgs []string
	// each project will be of the format "orgID/projectID"
	Projects []string
}

type GitspaceInstanceFilter struct {
	UserIdentifier      string
	LastUsedBefore      int64
	LastHeartBeatBefore int64
	LastUpdatedBefore   int64
	States              []enum.GitspaceInstanceStateType
	SpaceIDs            []int64
	// AllowAllSpaces is enabled for cde-manager jobs to list all gitspaces
	AllowAllSpaces bool
	Limit          int
}

func (g *GitspaceInstance) GetGitspaceState() (enum.GitspaceStateType, error) {
	if g == nil {
		return enum.GitspaceStateError, fmt.Errorf("GitspaceInstance is nil")
	}
	instanceState := g.State
	//nolint:exhaustive
	switch instanceState {
	case enum.GitspaceInstanceStateRunning:
		return enum.GitspaceStateRunning, nil
	case enum.GitspaceInstanceStateStopped:
		return enum.GitspaceStateStopped, nil
	case emptyGitspaceInstanceState,
		enum.GitspaceInstanceStateUninitialized,
		enum.GitspaceInstanceStateDeleted,
		enum.GitspaceInstanceStateCleaned:
		return enum.GitspaceStateUninitialized, nil
	case enum.GitspaceInstanceStateError,
		enum.GitspaceInstanceStateUnknown:
		return enum.GitspaceStateError, nil
	case enum.GitspaceInstanceStateStarting:
		return enum.GitspaceStateStarting, nil
	case enum.GitspaceInstanceStateStopping:
		return enum.GitspaceStateStopping, nil
	case enum.GitSpaceInstanceStateCleaning,
		enum.GitSpaceInstanceStateResetting:
		return enum.GitSpaceStateCleaning, nil
	default:
		return enum.GitspaceStateError, fmt.Errorf("unsupported gitspace instance state %s", string(instanceState))
	}
}
