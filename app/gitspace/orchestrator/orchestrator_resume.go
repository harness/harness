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
	"strconv"
	"time"

	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

// ResumeStartGitspace saves the provisioned infra, resolves the code repo details & creates the Gitspace container.
func (o Orchestrator) ResumeStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	provisionedInfra types.Infrastructure,
) (types.GitspaceInstance, *types.GitspaceError) {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateStarting
	secretValue, err := utils.ResolveSecret(ctx, o.secretResolverFactory, gitspaceConfig)
	if err != nil {
		return *gitspaceInstance, &types.GitspaceError{
			Error: fmt.Errorf("cannot resolve secret for ID %s: %w",
				gitspaceConfig.InfraProviderResource.UID, err),
			ErrorMessage: ptr.String(err.Error()),
		}
	}
	gitspaceInstance.AccessKey = secretValue

	ideSvc, err := o.ideFactory.GetIDE(gitspaceConfig.IDE)
	if err != nil {
		return *gitspaceInstance, &types.GitspaceError{
			Error:        err,
			ErrorMessage: ptr.String(err.Error()),
		}
	}

	err = o.infraProvisioner.PostInfraEventComplete(ctx, gitspaceConfig, provisionedInfra, enum.InfraEventProvision)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return *gitspaceInstance, &types.GitspaceError{
			Error: fmt.Errorf("cannot provision infrastructure for ID %s: %w",
				gitspaceConfig.InfraProviderResource.UID, err),
			ErrorMessage: ptr.String(err.Error()),
		}
	}

	if provisionedInfra.Status != enum.InfraStatusProvisioned {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)
		infraStateErr := fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			provisionedInfra.Status,
			enum.InfraStatusProvisioned,
			gitspaceConfig.GitspaceInstance.Identifier,
		)
		return *gitspaceInstance, &types.GitspaceError{
			Error:        infraStateErr,
			ErrorMessage: ptr.String(infraStateErr.Error()),
		}
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningCompleted)

	scmResolvedDetails, err := o.scm.GetSCMRepoDetails(ctx, gitspaceConfig)
	if err != nil {
		return *gitspaceInstance, &types.GitspaceError{
			Error: fmt.Errorf("failed to fetch code repo details for gitspace config ID %d: %w",
				gitspaceConfig.ID, err),
			ErrorMessage: ptr.String(err.Error()),
		}
	}
	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	containerOrchestrator, err := o.containerOrchestratorFactory.GetContainerOrchestrator(provisionedInfra.ProviderType)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)
		return *gitspaceInstance, &types.GitspaceError{
			Error: fmt.Errorf("failed to get the container orchestrator for infra provider type %s: %w",
				provisionedInfra.ProviderType, err),
			ErrorMessage: ptr.String(err.Error()),
		}
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationStart)

	// fetch connector information and send details to gitspace agent
	gitspaceSpecs := scmResolvedDetails.DevcontainerConfig.Customizations.ExtractGitspaceSpec()
	connectorRefs := getConnectorRefs(gitspaceSpecs)
	if len(connectorRefs) > 0 {
		connectors, err := o.platformConnector.FetchConnectors(ctx, connectorRefs, gitspaceConfig.SpacePath)
		if err != nil {
			fetchConnectorErr := fmt.Errorf("failed to fetch connectors for gitspace: %v :%w",
				connectorRefs,
				err,
			)
			return *gitspaceInstance, &types.GitspaceError{
				Error:        fetchConnectorErr,
				ErrorMessage: ptr.String(fetchConnectorErr.Error()),
			}
		}
		gitspaceConfig.Connectors = connectors
	}

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser

	err = containerOrchestrator.CreateAndStartGitspace(
		ctx, gitspaceConfig, provisionedInfra, *scmResolvedDetails, o.config.DefaultBaseImage, ideSvc)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationFailed)

		return *gitspaceInstance, &types.GitspaceError{
			Error:        fmt.Errorf("couldn't call the agent start API: %w", err),
			ErrorMessage: ptr.String(err.Error()), // TODO: Fetch explicit error msg from container orchestrator
		}
	}

	return *gitspaceConfig.GitspaceInstance, nil
}

// FinishResumeStartGitspace needs to be called from the API Handler.
func (o Orchestrator) FinishResumeStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	provisionedInfra types.Infrastructure,
	startResponse *response.StartResponse,
) (types.GitspaceInstance, *types.GitspaceError) {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	if startResponse == nil || startResponse.Status == response.FailureStatus {
		gitspaceInstance.State = enum.GitspaceInstanceStateError
		err := fmt.Errorf("gitspace agent does not specify the error for failure")
		if startResponse != nil && startResponse.ErrMessage != "" {
			err = fmt.Errorf("%s", startResponse.ErrMessage)
		}
		return *gitspaceInstance, &types.GitspaceError{
			Error:        err,
			ErrorMessage: ptr.String(err.Error()),
		}
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationCompleted)

	ideSvc, err := o.ideFactory.GetIDE(gitspaceConfig.IDE)
	if err != nil {
		return *gitspaceInstance, &types.GitspaceError{
			Error:        err,
			ErrorMessage: ptr.String(err.Error()),
		}
	}

	ideURLString := generateIDEURL(provisionedInfra, ideSvc, startResponse)
	gitspaceInstance.URL = &ideURLString

	now := time.Now().UnixMilli()
	gitspaceInstance.LastUsed = &now
	gitspaceInstance.ActiveTimeStarted = &now
	gitspaceInstance.LastHeartbeat = &now
	gitspaceInstance.State = enum.GitspaceInstanceStateRunning

	if gitspaceConfig.IsMarkedForReset {
		gitspaceConfig.IsMarkedForReset = false
		err := o.gitspaceConfigStore.Update(ctx, &gitspaceConfig)
		if err != nil {
			return *gitspaceInstance, &types.GitspaceError{
				Error:        err,
				ErrorMessage: ptr.String(err.Error()),
			}
		}
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStartCompleted)
	return *gitspaceInstance, nil
}

func generateIDEURL(
	provisionedInfra types.Infrastructure,
	ideSvc ide.IDE,
	startResponse *response.StartResponse,
) string {
	idePort := ideSvc.Port()
	var forwardedPort string

	if provisionedInfra.GitspacePortMappings[idePort.Port].PublishedPort == 0 {
		forwardedPort = startResponse.PublishedPorts[idePort.Port]
	} else {
		forwardedPort = strconv.Itoa(provisionedInfra.GitspacePortMappings[idePort.Port].ForwardedPort)
	}

	host := provisionedInfra.GitspaceHost
	if provisionedInfra.ProxyGitspaceHost != "" {
		host = provisionedInfra.ProxyGitspaceHost
	}

	return ideSvc.GenerateURL(startResponse.AbsoluteRepoPath, host, forwardedPort, startResponse.RemoteUser)
}

// ResumeStopGitspace saves the deprovisioned infra details.
func (o Orchestrator) ResumeStopGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	stoppedInfra types.Infrastructure,
) (enum.GitspaceInstanceStateType, *types.GitspaceError) {
	instanceState := enum.GitspaceInstanceStateError

	err := o.infraProvisioner.PostInfraEventComplete(ctx, gitspaceConfig, stoppedInfra, enum.InfraEventStop)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)
		infraStopErr := fmt.Errorf("cannot stop provisioned infrastructure with ID %s: %w",
			gitspaceConfig.InfraProviderResource.UID, err)
		return instanceState, &types.GitspaceError{
			Error:        infraStopErr,
			ErrorMessage: ptr.String(infraStopErr.Error()), // TODO: Fetch explicit error msg
		}
	}

	if stoppedInfra.Status != enum.InfraStatusDestroyed &&
		stoppedInfra.Status != enum.InfraStatusStopped {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopFailed)
		incorrectInfraStateErr := fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			stoppedInfra.Status,
			enum.InfraStatusDestroyed,
			gitspaceConfig.GitspaceInstance.Identifier)
		return instanceState, &types.GitspaceError{
			Error:        incorrectInfraStateErr,
			ErrorMessage: ptr.String(incorrectInfraStateErr.Error()), // TODO: Fetch explicit error msg
		}
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraStopCompleted)

	instanceState = enum.GitspaceInstanceStateStopped

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStopCompleted)

	return instanceState, nil
}

// ResumeDeleteGitspace saves the deprovisioned infra details.
func (o Orchestrator) ResumeDeleteGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	deprovisionedInfra types.Infrastructure,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	err := o.infraProvisioner.PostInfraEventComplete(ctx, gitspaceConfig, deprovisionedInfra, enum.InfraEventDeprovision)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraDeprovisioningFailed)
		return instanceState, fmt.Errorf(
			"cannot deprovision infrastructure with ID %s: %w", gitspaceConfig.InfraProviderResource.UID, err)
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

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionResetCompleted)
	return instanceState, nil
}

// ResumeCleanupInstanceResources saves the cleaned up infra details.
func (o Orchestrator) ResumeCleanupInstanceResources(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	cleanedUpInfra types.Infrastructure,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	err := o.infraProvisioner.PostInfraEventComplete(ctx, gitspaceConfig, cleanedUpInfra, enum.InfraEventCleanup)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraCleanupFailed)

		return instanceState, fmt.Errorf(
			"cannot cleanup provisioned infrastructure with ID %s: %w",
			gitspaceConfig.InfraProviderResource.UID,
			err,
		)
	}

	if cleanedUpInfra.Status != enum.InfraStatusDestroyed && cleanedUpInfra.Status != enum.InfraStatusStopped {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraCleanupFailed)

		return instanceState, fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			cleanedUpInfra.Status,
			[]enum.InfraStatus{enum.InfraStatusDestroyed, enum.InfraStatusStopped},
			gitspaceConfig.GitspaceInstance.Identifier)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraCleanupCompleted)

	instanceState = enum.GitspaceInstanceStateCleaned

	return instanceState, nil
}

func getConnectorRefs(specs *types.GitspaceCustomizationSpecs) []string {
	if specs == nil {
		return nil
	}
	log.Debug().Msgf("Customization connectors: %v", specs.Connectors)
	var connectorRefs []string
	for _, connector := range specs.Connectors {
		connectorRefs = append(connectorRefs, connector.ID)
	}

	return connectorRefs
}
