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
	"time"

	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/gitspace/secret"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

const harnessUser = "harness"

type Config struct {
	DefaultBaseImage string
}

type orchestrator struct {
	scm                        *scm.SCM
	infraProviderResourceStore store.InfraProviderResourceStore
	infraProvisioner           infrastructure.InfraProvisioner
	containerOrchestrator      container.Orchestrator
	eventReporter              *events.Reporter
	config                     *Config
	vsCodeService              *ide.VSCode
	vsCodeWebService           *ide.VSCodeWeb
	secretResolverFactory      *secret.ResolverFactory
}

var _ Orchestrator = (*orchestrator)(nil)

func NewOrchestrator(
	scm *scm.SCM,
	infraProviderResourceStore store.InfraProviderResourceStore,
	infraProvisioner infrastructure.InfraProvisioner,
	containerOrchestrator container.Orchestrator,
	eventReporter *events.Reporter,
	config *Config,
	vsCodeService *ide.VSCode,
	vsCodeWebService *ide.VSCodeWeb,
	secretResolverFactory *secret.ResolverFactory,
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
		secretResolverFactory:      secretResolverFactory,
	}
}

func (o orchestrator) TriggerStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) error {
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerStart)
	scmResolvedDetails, err := o.scm.GetSCMRepoDetails(ctx, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerFailed)
		return fmt.Errorf(
			"failed to fetch code repo details for gitspace config ID %w %d", err, gitspaceConfig.ID)
	}
	devcontainerConfig := scmResolvedDetails.DevcontainerConfig

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerCompleted)

	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig, devcontainerConfig)
	if err != nil {
		return fmt.Errorf("cannot get the ports required for gitspace during start: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningStart)

	err = o.infraProvisioner.TriggerProvision(ctx, gitspaceConfig, requiredGitspacePorts)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return fmt.Errorf(
			"cannot trigger provision infrastructure for ID %s: %w", gitspaceConfig.InfraProviderResource.UID, err)
	}

	return nil
}

func (o orchestrator) TriggerStopGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) error {
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig,
		[]enum.InfraStatus{enum.InfraStatusProvisioned, enum.InfraStatusStopped})
	if err != nil {
		return fmt.Errorf(
			"unable to find provisioned infra while triggering stop for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}
	if gitspaceConfig.GitspaceInstance.State == enum.GitspaceInstanceStateRunning {
		err = o.stopGitspaceContainer(ctx, gitspaceConfig, *infra)
	}
	if err != nil {
		return err
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopStart)

	err = o.infraProvisioner.TriggerStop(ctx, gitspaceConfig.InfraProviderResource, *infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)

		return fmt.Errorf(
			"cannot trigger stop infrastructure with ID %s: %w", gitspaceConfig.InfraProviderResource.UID, err)
	}

	return nil
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

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser

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

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser

	err = o.containerOrchestrator.StopAndRemoveGitspace(ctx, gitspaceConfig, infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionFailed)
		log.Err(err).Msgf("error stopping the Gitspace container")
	} else {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionCompleted)
	}
	return nil
}

func (o orchestrator) TriggerCleanupInstanceResources(ctx context.Context, gitspaceConfig types.GitspaceConfig) error {
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig,
		[]enum.InfraStatus{
			enum.InfraStatusProvisioned,
			enum.InfraStatusStopped,
			enum.InfraStatusPending,
			enum.InfraStatusUnknown,
			enum.InfraStatusDestroyed,
		})
	if err != nil {
		return fmt.Errorf(
			"unable to find provisioned infra while triggering cleanup for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}

	if gitspaceConfig.GitspaceInstance.State != enum.GitSpaceInstanceStateCleaning {
		return fmt.Errorf("cannot trigger cleanup, expected state: %s, actual state: %s ",
			enum.GitSpaceInstanceStateCleaning,
			gitspaceConfig.GitspaceInstance.State,
		)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraCleanupStart)

	err = o.infraProvisioner.TriggerCleanupInstance(ctx, gitspaceConfig, *infra)
	if err != nil {
		return fmt.Errorf("cannot trigger cleanup infrastructure with ID %s: %w",
			gitspaceConfig.InfraProviderResource.UID,
			err,
		)
	}

	return nil
}

func (o orchestrator) TriggerDeleteGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	canDeleteUserData bool,
) error {
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig,
		[]enum.InfraStatus{
			enum.InfraStatusProvisioned,
			enum.InfraStatusStopped,
			enum.InfraStatusDestroyed,
			enum.InfraStatusError,
			enum.InfraStatusUnknown,
		})
	if err != nil {
		return fmt.Errorf(
			"unable to find provisioned infra while triggering delete for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}
	if infra.ProviderType == enum.InfraProviderTypeDocker {
		if err = o.stopAndRemoveGitspaceContainer(ctx, gitspaceConfig, *infra); err != nil {
			return err
		}
	}
	if canDeleteUserData {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningStart)
	} else {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraResetStart)
	}

	err = o.infraProvisioner.TriggerDeprovision(ctx, gitspaceConfig, *infra, canDeleteUserData)
	if err != nil {
		if canDeleteUserData {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)
		} else {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraResetFailed)
		}
		return fmt.Errorf(
			"cannot trigger deprovision infrastructure with ID %s: %w", gitspaceConfig.InfraProviderResource.UID, err)
	}

	return nil
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

func (o orchestrator) getPortsRequiredForGitspace(
	gitspaceConfig types.GitspaceConfig,
	devcontainerConfig types.DevcontainerConfig,
) ([]types.GitspacePort, error) {
	// TODO: What if the required ports in the config have deviated from when the last instance was created?
	resolvedIDE, err := o.getIDEService(gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to get IDE service while checking required Gitspace ports: %w", err)
	}
	idePort := resolvedIDE.Port()
	gitspacePorts := []types.GitspacePort{*idePort}
	forwardPorts := container.ExtractForwardPorts(devcontainerConfig)

	for _, port := range forwardPorts {
		gitspacePorts = append(gitspacePorts, types.GitspacePort{
			Port:     port,
			Protocol: enum.CommunicationProtocolHTTP,
		})
	}
	return gitspacePorts, nil
}

func (o orchestrator) GetGitspaceLogs(ctx context.Context, gitspaceConfig types.GitspaceConfig) (string, error) {
	if gitspaceConfig.GitspaceInstance == nil {
		return "", fmt.Errorf("gitspace %s is not setup yet, please try later", gitspaceConfig.Identifier)
	}
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig, []enum.InfraStatus{enum.InfraStatusProvisioned})
	if err != nil {
		return "", fmt.Errorf(
			"unable to find provisioned infra while fetching logs for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser
	logs, err := o.containerOrchestrator.StreamLogs(ctx, gitspaceConfig, *infra)
	if err != nil {
		return "", fmt.Errorf("error while fetching logs from container orchestrator: %w", err)
	}

	return logs, nil
}

func (o orchestrator) getProvisionedInfra(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	expectedStatus []enum.InfraStatus,
) (*types.Infrastructure, error) {
	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig, types.DevcontainerConfig{})
	if err != nil {
		return nil, fmt.Errorf("cannot get the ports required for gitspace: %w", err)
	}

	infra, err := o.infraProvisioner.Find(ctx, gitspaceConfig, requiredGitspacePorts)
	if err != nil {
		return nil, fmt.Errorf("cannot find the provisioned infra: %w", err)
	}

	if !slices.Contains(expectedStatus, infra.Status) {
		return nil, fmt.Errorf("expected infra state in %v, actual state is: %s", expectedStatus, infra.Status)
	}

	if infra.Storage == "" {
		log.Warn().Msgf("couldn't find the storage for resource ID %s", gitspaceConfig.InfraProviderResource.UID)
	}

	return infra, nil
}
