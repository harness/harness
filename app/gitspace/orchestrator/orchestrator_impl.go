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

package orchestrator

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type orchestrator struct {
	scm                        scm.SCM
	infraProviderResourceStore store.InfraProviderResourceStore
	infraProvisioner           infrastructure.InfraProvisioner
	containerOrchestrator      container.Orchestrator
}

var _ Orchestrator = (*orchestrator)(nil)

func NewOrchestrator(
	scm scm.SCM,
	infraProviderResourceStore store.InfraProviderResourceStore,
	infraProvisioner infrastructure.InfraProvisioner,
	containerOrchestrator container.Orchestrator,
) Orchestrator {
	return orchestrator{
		scm:                        scm,
		infraProviderResourceStore: infraProviderResourceStore,
		infraProvisioner:           infraProvisioner,
		containerOrchestrator:      containerOrchestrator,
	}
}

func (o orchestrator) StartGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*types.GitspaceInstance, error) {
	devcontainerConfig, err := o.scm.DevcontainerConfig(ctx, gitspaceConfig)
	if err != nil {
		log.Warn().Err(err).Msg("devcontainerConfig fetch failed.")
	}

	if devcontainerConfig == nil {
		log.Warn().Err(err).Msg("devcontainerConfig is nil, using empty config")
		devcontainerConfig = &types.DevcontainerConfig{}
	}

	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return nil, fmt.Errorf("cannot get the infraProviderResource for ID %d: %w",
			gitspaceConfig.InfraProviderResourceID, err)
	}

	infra, err := o.infraProvisioner.Provision(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot provision infrastructure for ID %d: %w",
			gitspaceConfig.InfraProviderResourceID, err)
	}

	gitspaceInstance := gitspaceConfig.GitspaceInstance

	err = o.containerOrchestrator.Status(ctx, infra)
	gitspaceInstance.State = enum.GitspaceInstanceStateError
	if err != nil {
		return gitspaceInstance, fmt.Errorf("couldn't call the agent health API: %w", err)
	}

	startResponse, err := o.containerOrchestrator.StartGitspace(ctx, gitspaceConfig, devcontainerConfig, infra)
	if err != nil {
		return gitspaceInstance, fmt.Errorf("couldn't call the agent start API: %w", err)
	}

	repoName, err := o.scm.RepositoryName(ctx, gitspaceConfig)
	if err != nil {
		log.Warn().Err(err).Msg("failed to fetch repository name.")
	}

	port := startResponse.PortsUsed[gitspaceConfig.IDE]

	var ideURL url.URL

	if infra.Host == "" {
		// TODO: This fix does not cover all use-cases. Ideally, we need to read the host name on which this docker is
		// running and set it as the infra.Host. Remove once that change is done.
		infra.Host = "localhost"
	}

	if gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
		ideURL = url.URL{
			Scheme:   "http",
			Host:     infra.Host + ":" + port,
			RawQuery: "folder=/" + startResponse.WorkingDirectory + "/" + repoName,
		}
	} else if gitspaceConfig.IDE == enum.IDETypeVSCode {
		// TODO: the following user ID is hard coded and should be changed.
		ideURL = url.URL{
			Scheme: "vscode-remote",
			Host:   "", // Empty since we include the host and port in the path
			Path: fmt.Sprintf(
				"ssh-remote+%s@%s:%s/%s/%s",
				"harness",
				infra.Host,
				port,
				startResponse.WorkingDirectory,
				repoName,
			),
		}
	}
	ideURLString := ideURL.String()
	gitspaceInstance.URL = &ideURLString

	gitspaceInstance.LastUsed = time.Now().UnixMilli()
	gitspaceInstance.State = enum.GitspaceInstanceStateRunning

	return gitspaceInstance, nil
}

func (o orchestrator) StopGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*types.GitspaceInstance, error) {
	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot get the infraProviderResource with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	infra, err := o.infraProvisioner.Find(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot find the provisioned infra: %w", err)
	}

	err = o.containerOrchestrator.StopGitspace(ctx, gitspaceConfig, infra)
	if err != nil {
		return nil, fmt.Errorf("error stopping the Gitspace container: %w", err)
	}

	_, err = o.infraProvisioner.Stop(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot stop provisioned infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	gitspaceInstance := gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateDeleted
	return gitspaceInstance, err
}

func (o orchestrator) DeleteGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*types.GitspaceInstance, error) {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot get the infraProviderResource with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}
	if gitspaceInstance.State == enum.GitspaceInstanceStateRunning ||
		gitspaceInstance.State == enum.GitspaceInstanceStateUnknown {
		infra, err := o.infraProvisioner.Find(ctx, infraProviderResource, gitspaceConfig)
		if err != nil {
			return nil, fmt.Errorf("cannot find the provisioned infra: %w", err)
		}

		err = o.containerOrchestrator.StopGitspace(ctx, gitspaceConfig, infra)
		if err != nil {
			return nil, fmt.Errorf("error stopping the Gitspace container: %w", err)
		}
	}
	_, err = o.infraProvisioner.Unprovision(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot stop provisioned infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}
	gitspaceInstance.State = enum.GitspaceInstanceStateDeleted
	return gitspaceInstance, nil
}
