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
	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/platformconnector"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/gitspace/secret"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

const harnessUser = "harness"

type Config struct {
	DefaultBaseImage string
}

type Orchestrator struct {
	scm                          *scm.SCM
	platformConnector            platformconnector.PlatformConnector
	infraProvisioner             infrastructure.InfraProvisioner
	containerOrchestratorFactory container.Factory
	eventReporter                *events.Reporter
	config                       *Config
	ideFactory                   ide.Factory
	secretResolverFactory        *secret.ResolverFactory
	gitspaceInstanceStore        store.GitspaceInstanceStore
	gitspaceConfigStore          store.GitspaceConfigStore
}

func NewOrchestrator(
	scm *scm.SCM,
	platformConnector platformconnector.PlatformConnector,
	infraProvisioner infrastructure.InfraProvisioner,
	containerOrchestratorFactory container.Factory,
	eventReporter *events.Reporter,
	config *Config,
	ideFactory ide.Factory,
	secretResolverFactory *secret.ResolverFactory,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	gitspaceConfigStore store.GitspaceConfigStore,
) Orchestrator {
	return Orchestrator{
		scm:                          scm,
		platformConnector:            platformConnector,
		infraProvisioner:             infraProvisioner,
		containerOrchestratorFactory: containerOrchestratorFactory,
		eventReporter:                eventReporter,
		config:                       config,
		ideFactory:                   ideFactory,
		secretResolverFactory:        secretResolverFactory,
		gitspaceInstanceStore:        gitspaceInstanceStore,
		gitspaceConfigStore:          gitspaceConfigStore,
	}
}

// TriggerStartGitspace fetches the infra resources configured for the gitspace and triggers the infra provisioning.
func (o Orchestrator) TriggerStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) *types.GitspaceError {
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerStart)
	scmResolvedDetails, err := o.scm.GetSCMRepoDetails(ctx, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerFailed)
		return &types.GitspaceError{
			Error: fmt.Errorf("failed to fetch code repo details for gitspace config ID %d: %w",
				gitspaceConfig.ID, err),
			ErrorMessage: ptr.String(err.Error()),
		}
	}
	devcontainerConfig := scmResolvedDetails.DevcontainerConfig
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerCompleted)
	gitspaceSpecs := devcontainerConfig.Customizations.ExtractGitspaceSpec()
	connectorRefs := getConnectorRefs(gitspaceSpecs)
	if len(connectorRefs) > 0 {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchConnectorsDetailsStart)
		connectors, err := o.platformConnector.FetchConnectors(
			ctx, connectorRefs, gitspaceConfig.SpacePath)
		if err != nil {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchConnectorsDetailsFailed)
			log.Ctx(ctx).Err(err).Msgf("failed to fetch connectors for gitspace: %v",
				connectorRefs,
			)
			return &types.GitspaceError{
				Error: fmt.Errorf("failed to fetch connectors for gitspace: %v :%w",
					connectorRefs,
					err,
				),
				ErrorMessage: ptr.String(err.Error()),
			}
		}
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchConnectorsDetailsCompleted)
		gitspaceConfig.Connectors = connectors
	}

	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig, devcontainerConfig)
	if err != nil {
		err = fmt.Errorf("cannot get the ports required for gitspace during start: %w", err)
		return &types.GitspaceError{
			Error:        err,
			ErrorMessage: ptr.String(err.Error()),
		}
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningStart)

	opts := infrastructure.InfraEventOpts{RequiredGitspacePorts: requiredGitspacePorts}
	err = o.infraProvisioner.TriggerInfraEventWithOpts(ctx, enum.InfraEventProvision, gitspaceConfig, nil, opts)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)
		return &types.GitspaceError{
			Error: fmt.Errorf("cannot trigger provision infrastructure for ID %s: %w",
				gitspaceConfig.InfraProviderResource.UID, err),
			ErrorMessage: ptr.String(err.Error()), // TODO: Fetch explicit error msg from infra provisioner.
		}
	}

	return nil
}

// TriggerStopGitspace stops the Gitspace container and triggers infra deprovisioning to deprovision
// all the infra resources which are not required to restart the Gitspace.
func (o Orchestrator) TriggerStopGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) *types.GitspaceError {
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig,
		[]enum.InfraStatus{enum.InfraStatusProvisioned, enum.InfraStatusStopped})
	if err != nil {
		infraNotFoundErr := fmt.Errorf("unable to find provisioned infra while triggering stop for gitspace "+
			"instance %s: %w", gitspaceConfig.GitspaceInstance.Identifier, err)
		return &types.GitspaceError{
			Error:        infraNotFoundErr,
			ErrorMessage: ptr.String(infraNotFoundErr.Error()), // TODO: Fetch explicit error msg
		}
	}
	if gitspaceConfig.GitspaceInstance.State == enum.GitspaceInstanceStateRunning ||
		gitspaceConfig.GitspaceInstance.State == enum.GitspaceInstanceStateStopping {
		err = o.stopGitspaceContainer(ctx, gitspaceConfig, *infra)
		if err != nil {
			return &types.GitspaceError{
				Error:        err,
				ErrorMessage: ptr.String(err.Error()), // TODO: Fetch explicit error msg
			}
		}
	}

	return nil
}

func (o Orchestrator) stopGitspaceContainer(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	containerOrchestrator, err := o.containerOrchestratorFactory.GetContainerOrchestrator(infra.ProviderType)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)
		return fmt.Errorf("couldn't get the container orchestrator: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceStopStart)

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser

	err = containerOrchestrator.StopGitspace(ctx, gitspaceConfig, infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceStopFailed)

		return fmt.Errorf("error stopping the Gitspace container: %w", err)
	}

	return nil
}

func (o Orchestrator) FinishStopGitspaceContainer(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	stopResponse *response.StopResponse,
) *types.GitspaceError {
	if stopResponse == nil || stopResponse.Status == response.FailureStatus {
		err := fmt.Errorf("gitspace agent does not specify the error for failure")
		if stopResponse != nil && stopResponse.ErrMessage != "" {
			err = fmt.Errorf("%s", stopResponse.ErrMessage)
		}
		return &types.GitspaceError{
			Error:        err,
			ErrorMessage: ptr.String(err.Error()),
		}
	}
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceStopCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopStart)

	err := o.infraProvisioner.TriggerInfraEvent(ctx, enum.InfraEventStop, gitspaceConfig, &infra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)
		infraStopErr := fmt.Errorf("cannot trigger stop infrastructure with ID %s: %w",
			gitspaceConfig.InfraProviderResource.UID, err)
		return &types.GitspaceError{
			Error:        infraStopErr,
			ErrorMessage: ptr.String(infraStopErr.Error()), // TODO: Fetch explicit error msg
		}
	}
	return nil
}

func (o Orchestrator) stopAndRemoveGitspaceContainer(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	canDeleteUserData bool,
) error {
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	containerOrchestrator, err := o.containerOrchestratorFactory.GetContainerOrchestrator(infra.ProviderType)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)
		return fmt.Errorf("couldn't get the container orchestrator: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionStart)

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser

	err = containerOrchestrator.RemoveGitspace(ctx, gitspaceConfig, infra, canDeleteUserData)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionFailed)
		log.Err(err).Msgf("error stopping the Gitspace container")
	}
	return nil
}

func (o Orchestrator) FinishStopAndRemoveGitspaceContainer(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	deleteResponse *response.DeleteResponse,
) *types.GitspaceError {
	if deleteResponse == nil || deleteResponse.Status == response.FailureStatus {
		err := fmt.Errorf("gitspace agent does not specify the error for failure")
		if deleteResponse != nil && deleteResponse.ErrMessage != "" {
			err = fmt.Errorf("%s", deleteResponse.ErrMessage)
		}
		return &types.GitspaceError{
			Error:        err,
			ErrorMessage: ptr.String(err.Error()),
		}
	}
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceDeletionCompleted)
	if deleteResponse.CanDeleteUserData {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningStart)
	} else {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraResetStart)
	}

	opts := infrastructure.InfraEventOpts{CanDeleteUserData: deleteResponse.CanDeleteUserData}
	err := o.infraProvisioner.TriggerInfraEventWithOpts(ctx, enum.InfraEventDeprovision, gitspaceConfig, &infra, opts)
	if err != nil {
		if deleteResponse.CanDeleteUserData {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)
		} else {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraResetFailed)
		}
		return &types.GitspaceError{
			Error: fmt.Errorf(
				"cannot trigger deprovision infrastructure with ID %s: %w",
				gitspaceConfig.InfraProviderResource.UID,
				err,
			),
			ErrorMessage: ptr.String(err.Error()),
		}
	}
	return nil
}

// TriggerCleanupInstanceResources cleans up all the resources exclusive to gitspace instance.
func (o Orchestrator) TriggerCleanupInstanceResources(ctx context.Context, gitspaceConfig types.GitspaceConfig) error {
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

	err = o.infraProvisioner.TriggerInfraEvent(ctx, enum.InfraEventCleanup, gitspaceConfig, infra)
	if err != nil {
		return fmt.Errorf("cannot trigger cleanup infrastructure with ID %s: %w",
			gitspaceConfig.InfraProviderResource.UID,
			err,
		)
	}

	return nil
}

// TriggerStopAndDeleteGitspace removes the Gitspace container and triggers infra deprovisioning to deprovision
// the infra resources.
// canDeleteUserData = false -> trigger deprovision of all resources except storage associated to user data.
// canDeleteUserData = true -> trigger deprovision of all resources.
func (o Orchestrator) TriggerStopAndDeleteGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	canDeleteUserData bool,
) error {
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig,
		[]enum.InfraStatus{
			enum.InfraStatusPending,
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
	if err = o.stopAndRemoveGitspaceContainer(ctx, gitspaceConfig, *infra, canDeleteUserData); err != nil {
		log.Warn().Msgf("error stopping and removing gitspace container: %v", err)
		// If stop fails, delete the gitspace anyway
		return o.triggerDeleteGitspace(ctx, gitspaceConfig, infra, canDeleteUserData)
	}

	// TODO: Add a job for cleanup of infra if stop fails
	log.Warn().Msgf(
		"Checking and force deleting the infra if required for gitspace instance %s",
		gitspaceConfig.GitspaceInstance.Identifier,
	)
	if err = o.waitForGitspaceCleanupOrTimeout(ctx, gitspaceConfig, 15*time.Minute, 60*time.Second); err != nil {
		return o.triggerDeleteGitspace(ctx, gitspaceConfig, infra, canDeleteUserData)
	}
	return nil
}

func (o Orchestrator) triggerDeleteGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra *types.Infrastructure,
	canDeleteUserData bool,
) error {
	opts := infrastructure.InfraEventOpts{CanDeleteUserData: true}
	err := o.infraProvisioner.TriggerInfraEventWithOpts(
		ctx,
		enum.InfraEventDeprovision,
		gitspaceConfig,
		infra,
		opts,
	)
	if err != nil {
		if canDeleteUserData {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)
		} else {
			o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraResetFailed)
		}
		return fmt.Errorf(
			"cannot trigger deprovision infrastructure with gitspace identifier %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier,
			err,
		)
	}
	return nil
}

func (o Orchestrator) waitForGitspaceCleanupOrTimeout(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	timeoutDuration time.Duration,
	tickInterval time.Duration,
) error {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	timeout := time.After(timeoutDuration)

	for {
		select {
		case <-ticker.C:
			instance, err := o.gitspaceInstanceStore.Find(ctx, gitspaceConfig.GitspaceInstance.ID)
			if err != nil {
				return fmt.Errorf(
					"failed to find gitspace instance %s: %w",
					gitspaceConfig.GitspaceInstance.Identifier,
					err,
				)
			}
			if instance.State == enum.GitspaceInstanceStateDeleted ||
				instance.State == enum.GitspaceInstanceStateCleaned {
				return nil
			}
		case <-timeout:
			return fmt.Errorf(
				"timeout waiting for gitspace cleanup for instance %s",
				gitspaceConfig.GitspaceInstance.Identifier,
			)
		}
	}
}

func (o Orchestrator) emitGitspaceEvent(
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

func (o Orchestrator) getPortsRequiredForGitspace(
	gitspaceConfig types.GitspaceConfig,
	devcontainerConfig types.DevcontainerConfig,
) ([]types.GitspacePort, error) {
	// TODO: What if the required ports in the config have deviated from when the last instance was created?
	resolvedIDE, err := o.ideFactory.GetIDE(gitspaceConfig.IDE)
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

// GetGitspaceLogs fetches gitspace's start/stop logs.
func (o Orchestrator) GetGitspaceLogs(ctx context.Context, gitspaceConfig types.GitspaceConfig) (string, error) {
	if gitspaceConfig.GitspaceInstance == nil {
		return "", fmt.Errorf("gitspace %s is not setup yet, please try later", gitspaceConfig.Identifier)
	}
	infra, err := o.getProvisionedInfra(ctx, gitspaceConfig, []enum.InfraStatus{enum.InfraStatusProvisioned})
	if err != nil {
		return "", fmt.Errorf(
			"unable to find provisioned infra while fetching logs for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}

	containerOrchestrator, err := o.containerOrchestratorFactory.GetContainerOrchestrator(infra.ProviderType)
	if err != nil {
		return "", fmt.Errorf("couldn't get the container orchestrator: %w", err)
	}

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser
	logs, err := containerOrchestrator.StreamLogs(ctx, gitspaceConfig, *infra)
	if err != nil {
		return "", fmt.Errorf("error while fetching logs from container orchestrator: %w", err)
	}

	return logs, nil
}

func (o Orchestrator) getProvisionedInfra(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	expectedStatus []enum.InfraStatus,
) (*types.Infrastructure, error) {
	infra, err := o.infraProvisioner.Find(ctx, gitspaceConfig)
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
