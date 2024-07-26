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
	"path/filepath"
	"strconv"
	"time"

	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type Config struct {
	DefaultBaseImage string
}

type orchestrator struct {
	scm                        scm.SCM
	infraProviderResourceStore store.InfraProviderResourceStore
	infraProvisioner           infrastructure.InfraProvisioner
	containerOrchestrator      container.Orchestrator
	eventReporter              *events.Reporter
	config                     *Config
	vsCodeService              *ide.VSCode
	vsCodeWebService           *ide.VSCodeWeb
}

var _ Orchestrator = (*orchestrator)(nil)

func NewOrchestrator(
	scm scm.SCM,
	infraProviderResourceStore store.InfraProviderResourceStore,
	infraProvisioner infrastructure.InfraProvisioner,
	containerOrchestrator container.Orchestrator,
	eventReporter *events.Reporter,
	config *Config,
	vsCodeService *ide.VSCode,
	vsCodeWebService *ide.VSCodeWeb,
) Orchestrator {
	return orchestrator{
		scm:                        scm,
		infraProviderResourceStore: infraProviderResourceStore,
		infraProvisioner:           infraProvisioner,
		containerOrchestrator:      containerOrchestrator,
		eventReporter:              eventReporter,
		config:                     config,
		vsCodeService:              vsCodeService,
		vsCodeWebService:           vsCodeWebService,
	}
}

func (o orchestrator) StartGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) error {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateError

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerStart)

	scmResolvedDetails, err := o.scm.Resolve(ctx, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerFailed)

		return fmt.Errorf("failed to fetch code repo details for gitspace config ID %w %d", err, gitspaceConfig.ID)
	}
	devcontainerConfig := scmResolvedDetails.DevcontainerConfig
	repoName := scmResolvedDetails.RepoName

	if devcontainerConfig == nil {
		log.Warn().Err(err).Msg("devcontainer config is nil, using empty config")
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerCompleted)

	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return fmt.Errorf("cannot get the infraprovider resource for ID %d: %w",
			gitspaceConfig.InfraProviderResourceID, err)
	}

	ideSvc, err := o.getIDEService(gitspaceConfig)
	if err != nil {
		return err
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningStart)

	idePort := ideSvc.Port()

	infra, err := o.infraProvisioner.Provision(ctx, infraProviderResource, gitspaceConfig, []int{idePort})
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return fmt.Errorf(
			"cannot provision infrastructure for ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	err = o.containerOrchestrator.Status(ctx, infra)
	gitspaceInstance.State = enum.GitspaceInstanceStateError
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)

		return fmt.Errorf("couldn't call the agent health API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationStart)

	startResponse, err := o.containerOrchestrator.CreateAndStartGitspace(
		ctx, gitspaceConfig, infra, scmResolvedDetails, o.config.DefaultBaseImage, ideSvc)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationFailed)

		return fmt.Errorf("couldn't call the agent start API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationCompleted)

	var ideURL url.URL

	var forwardedPort string

	if infra.PortMappings[idePort].PublishedPort == 0 {
		forwardedPort = startResponse.PublishedPorts[idePort]
	} else {
		forwardedPort = strconv.Itoa(infra.PortMappings[idePort].ForwardedPort)
	}

	if gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
		ideURL = url.URL{
			Scheme:   "http",
			Host:     infra.Host + ":" + forwardedPort,
			RawQuery: filepath.Join("folder=", repoName),
		}
	} else if gitspaceConfig.IDE == enum.IDETypeVSCode {
		// TODO: the following userID is hard coded and should be changed.
		userID := "harness"
		ideURL = url.URL{
			Scheme: "vscode-remote",
			Host:   "", // Empty since we include the host and port in the path
			Path: fmt.Sprintf(
				"ssh-remote+%s@%s:%s",
				userID,
				infra.Host,
				filepath.Join(forwardedPort, repoName),
			),
		}
	}
	ideURLString := ideURL.String()
	gitspaceInstance.URL = &ideURLString

	gitspaceInstance.LastUsed = time.Now().UnixMilli()
	gitspaceInstance.State = enum.GitspaceInstanceStateRunning

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStartCompleted)

	return nil
}

func (o orchestrator) StopGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) error {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateError

	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return fmt.Errorf(
			"cannot get the infraProviderResource with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	infra, err := o.infraProvisioner.Find(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		return fmt.Errorf("cannot find the provisioned infra: %w", err)
	}

	err = o.stopGitspaceContainer(ctx, gitspaceConfig, infra)
	if err != nil {
		return err
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopStart)

	_, err = o.infraProvisioner.Stop(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)

		return fmt.Errorf(
			"cannot stop provisioned infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopCompleted)

	gitspaceInstance.State = enum.GitspaceInstanceStateDeleted

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStopCompleted)

	return err
}

func (o orchestrator) stopGitspaceContainer(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	infra *infraprovider.Infrastructure,
) error {
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	err := o.containerOrchestrator.Status(ctx, infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)

		return fmt.Errorf("couldn't call the agent health API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceStopStart)

	err = o.containerOrchestrator.StopGitspace(ctx, gitspaceConfig, infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceStopFailed)

		return fmt.Errorf("error stopping the Gitspace container: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceStopCompleted)
	return nil
}

func (o orchestrator) stopAndRemoveGitspaceContainer(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	infra *infraprovider.Infrastructure,
) error {
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	err := o.containerOrchestrator.Status(ctx, infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)

		return fmt.Errorf("couldn't call the agent health API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionStart)

	err = o.containerOrchestrator.StopAndRemoveGitspace(ctx, gitspaceConfig, infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionFailed)

		return fmt.Errorf("error stopping the Gitspace container: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionCompleted)
	return nil
}

func (o orchestrator) DeleteGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*types.GitspaceInstance, error) {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateError

	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot get the infraProviderResource with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	infra, err := o.infraProvisioner.Find(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot find the provisioned infra: %w", err)
	}

	err = o.stopAndRemoveGitspaceContainer(ctx, gitspaceConfig, infra)
	if err != nil {
		return nil, err
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningStart)

	_, err = o.infraProvisioner.Deprovision(ctx, infraProviderResource, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)

		return nil, fmt.Errorf(
			"cannot deprovision infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningCompleted)

	gitspaceInstance.State = enum.GitspaceInstanceStateDeleted

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStopCompleted)

	return gitspaceInstance, nil
}

func (o orchestrator) emitGitspaceEvent(
	ctx context.Context,
	config *types.GitspaceConfig,
	eventType enum.GitspaceEventType,
) {
	o.eventReporter.EmitGitspaceEvent(
		ctx,
		events.GitspaceEvent,
		&events.GitspaceEventPayload{
			QueryKey:   config.Identifier,
			EntityID:   config.GitspaceInstance.ID,
			EntityType: enum.GitspaceEntityTypeGitspaceInstance,
			EventType:  eventType,
			Timestamp:  time.Now().UnixNano(),
		})
}

func (o orchestrator) getIDEService(gitspaceConfig *types.GitspaceConfig) (ide.IDE, error) {
	var ideService ide.IDE

	switch gitspaceConfig.IDE {
	case enum.IDETypeVSCode:
		ideService = o.vsCodeService
	case enum.IDETypeVSCodeWeb:
		ideService = o.vsCodeWebService
	default:
		return nil, fmt.Errorf("unsupported IDE: %s", gitspaceConfig.IDE)
	}

	return ideService, nil
}
