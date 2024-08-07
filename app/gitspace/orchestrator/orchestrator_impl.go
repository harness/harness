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

func (o orchestrator) TriggerStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError
	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return instanceState, fmt.Errorf("cannot get the infraprovider resource for ID %d: %w",
			gitspaceConfig.InfraProviderResourceID, err)
	}

	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig)
	if err != nil {
		return instanceState, fmt.Errorf("cannot get the ports required for gitspace during start: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningStart)

	err = o.infraProvisioner.TriggerProvision(ctx, *infraProviderResource, gitspaceConfig, requiredGitspacePorts)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return instanceState, fmt.Errorf(
			"cannot trigger provision infrastructure for ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	instanceState = enum.GitspaceInstanceStateStarting

	return instanceState, nil
}

func (o orchestrator) TriggerStopGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return instanceState, fmt.Errorf(
			"cannot get the infraProviderResource with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig)
	if err != nil {
		return instanceState, fmt.Errorf("cannot get the ports required for gitspace during start: %w", err)
	}

	infra, err := o.infraProvisioner.Find(ctx, *infraProviderResource, gitspaceConfig, requiredGitspacePorts)
	if err != nil {
		return instanceState, fmt.Errorf("cannot find the provisioned infra: %w", err)
	}

	err = o.stopGitspaceContainer(ctx, gitspaceConfig, *infra)
	if err != nil {
		return instanceState, err
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopStart)

	err = o.infraProvisioner.TriggerStop(ctx, *infraProviderResource, *infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)

		return instanceState, fmt.Errorf(
			"cannot trigger stop infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	instanceState = enum.GitspaceInstanceStateStopping

	return instanceState, nil
}

func (o orchestrator) stopGitspaceContainer(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
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
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
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

func (o orchestrator) TriggerDeleteGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	infraProviderResource, err := o.infraProviderResourceStore.Find(ctx, gitspaceConfig.InfraProviderResourceID)
	if err != nil {
		return instanceState, fmt.Errorf(
			"cannot get the infraProviderResource with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig)
	if err != nil {
		return instanceState, fmt.Errorf("cannot get the ports required for gitspace during start: %w", err)
	}

	infra, err := o.infraProvisioner.Find(ctx, *infraProviderResource, gitspaceConfig, requiredGitspacePorts)
	if err != nil {
		return instanceState, fmt.Errorf("cannot find the provisioned infra: %w", err)
	}

	err = o.stopAndRemoveGitspaceContainer(ctx, gitspaceConfig, *infra)
	if err != nil {
		return instanceState, err
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningStart)

	err = o.infraProvisioner.TriggerDeprovision(ctx, *infraProviderResource, gitspaceConfig, *infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)

		return instanceState, fmt.Errorf(
			"cannot trigger deprovision infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}
	instanceState = enum.GitspaceInstanceStateStopping

	return instanceState, nil
}

func (o orchestrator) emitGitspaceEvent(
	ctx context.Context,
	config types.GitspaceConfig,
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

func (o orchestrator) getIDEService(gitspaceConfig types.GitspaceConfig) (ide.IDE, error) {
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

func (o orchestrator) ResumeStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	provisionedInfra types.Infrastructure,
) (types.GitspaceInstance, error) {
	gitspaceInstance := *gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateError

	ideSvc, err := o.getIDEService(gitspaceConfig)
	if err != nil {
		return gitspaceInstance, err
	}

	idePort := ideSvc.Port()

	err = o.infraProvisioner.ResumeProvision(ctx, gitspaceConfig, provisionedInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return gitspaceInstance, fmt.Errorf(
			"cannot provision infrastructure for ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	if provisionedInfra.Status != enum.InfraStatusProvisioned {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return gitspaceInstance, fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			provisionedInfra.Status,
			enum.InfraStatusProvisioned,
			gitspaceConfig.GitspaceInstance.Identifier)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerStart)

	scmResolvedDetails, err := o.scm.GetSCMRepoDetails(ctx, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerFailed)

		return gitspaceInstance, fmt.Errorf(
			"failed to fetch code repo details for gitspace config ID %w %d", err, gitspaceConfig.ID)
	}
	devcontainerConfig := scmResolvedDetails.DevcontainerConfig
	repoName := scmResolvedDetails.RepoName

	if devcontainerConfig == nil {
		log.Warn().Err(err).Msg("devcontainer config is nil, using empty config")
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	err = o.containerOrchestrator.Status(ctx, provisionedInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)

		return gitspaceInstance, fmt.Errorf("couldn't call the agent health API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationStart)

	startResponse, err := o.containerOrchestrator.CreateAndStartGitspace(
		ctx, gitspaceConfig, provisionedInfra, *scmResolvedDetails, o.config.DefaultBaseImage, ideSvc)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationFailed)

		return gitspaceInstance, fmt.Errorf("couldn't call the agent start API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationCompleted)

	var ideURL url.URL

	var forwardedPort string

	if provisionedInfra.GitspacePortMappings[idePort].PublishedPort == 0 {
		forwardedPort = startResponse.PublishedPorts[idePort]
	} else {
		forwardedPort = strconv.Itoa(provisionedInfra.GitspacePortMappings[idePort].ForwardedPort)
	}

	host := provisionedInfra.Host
	if provisionedInfra.ProxyHost != "" {
		host = provisionedInfra.ProxyHost
	}

	if gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
		ideURL = url.URL{
			Scheme:   "http",
			Host:     host + ":" + forwardedPort,
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
				host,
				filepath.Join(forwardedPort, repoName),
			),
		}
	}
	ideURLString := ideURL.String()
	gitspaceInstance.URL = &ideURLString

	gitspaceInstance.LastUsed = time.Now().UnixMilli()
	gitspaceInstance.State = enum.GitspaceInstanceStateRunning

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStartCompleted)

	return gitspaceInstance, nil
}

func (o orchestrator) ResumeStopGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	stoppedInfra types.Infrastructure,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	err := o.infraProvisioner.ResumeStop(ctx, gitspaceConfig, stoppedInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)

		return instanceState, fmt.Errorf(
			"cannot stop provisioned infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	if stoppedInfra.Status != enum.InfraStatusDestroyed {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)

		return instanceState, fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			stoppedInfra.Status,
			enum.InfraStatusDestroyed,
			gitspaceConfig.GitspaceInstance.Identifier)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopCompleted)

	instanceState = enum.GitspaceInstanceStateDeleted

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStopCompleted)

	return instanceState, nil
}

func (o orchestrator) ResumeDeleteGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	deprovisionedInfra types.Infrastructure,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	err := o.infraProvisioner.ResumeDeprovision(ctx, gitspaceConfig, deprovisionedInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)
		return instanceState, fmt.Errorf(
			"cannot deprovision infrastructure with ID %d: %w", gitspaceConfig.InfraProviderResourceID, err)
	}

	if deprovisionedInfra.Status != enum.InfraStatusDestroyed {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)

		return instanceState, fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			deprovisionedInfra.Status,
			enum.InfraStatusDestroyed,
			gitspaceConfig.GitspaceInstance.Identifier)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningCompleted)

	instanceState = enum.GitspaceInstanceStateDeleted

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStopCompleted)
	return instanceState, nil
}

func (o orchestrator) getPortsRequiredForGitspace(
	gitspaceConfig types.GitspaceConfig,
) ([]int, error) {
	// TODO: What if the required ports in the config have deviated from when the last instance was created?
	ideSvc, err := o.getIDEService(gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to get IDE service while checking required Gitspace ports: %w", err)
	}

	idePort := ideSvc.Port()
	return []int{idePort}, nil
}
