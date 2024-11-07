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
	"strings"
	"time"

	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/gitspace/secret"
	secretenum "github.com/harness/gitness/app/gitspace/secret/enum"
	"github.com/harness/gitness/app/paths"
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
	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig)
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

func (o orchestrator) ResumeCleanupInstanceResources(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	cleanedUpInfra types.Infrastructure,
) (enum.GitspaceInstanceStateType, error) {
	instanceState := enum.GitspaceInstanceStateError

	err := o.infraProvisioner.ResumeCleanupInstance(ctx, gitspaceConfig, cleanedUpInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraCleanupFailed)

		return instanceState, fmt.Errorf(
			"cannot clenup provisioned infrastructure with ID %s: %w",
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

func (o orchestrator) ResumeStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	provisionedInfra types.Infrastructure,
) (types.GitspaceInstance, error) {
	gitspaceInstance := gitspaceConfig.GitspaceInstance
	gitspaceInstance.State = enum.GitspaceInstanceStateError

	secretResolver, err := o.getSecretResolver(gitspaceInstance.AccessType)
	if err != nil {
		log.Err(err).Msgf("could not find secret resolver for type: %s", gitspaceInstance.AccessType)
		return *gitspaceInstance, err
	}
	rootSpaceID, _, err := paths.DisectRoot(gitspaceConfig.SpacePath)
	if err != nil {
		log.Err(err).Msgf("unable to find root space id from space path: %s", gitspaceConfig.SpacePath)
		return *gitspaceInstance, err
	}
	resolvedSecret, err := secretResolver.Resolve(ctx, secret.ResolutionContext{
		UserIdentifier:     gitspaceConfig.GitspaceUser.Identifier,
		GitspaceIdentifier: gitspaceConfig.Identifier,
		SecretRef:          *gitspaceInstance.AccessKeyRef,
		SpaceIdentifier:    rootSpaceID,
	})
	if err != nil {
		log.Err(err).Msgf("could not resolve secret type: %s, ref: %s",
			gitspaceInstance.AccessType, *gitspaceInstance.AccessKeyRef)
		return *gitspaceInstance, err
	}
	gitspaceInstance.AccessKey = &resolvedSecret.SecretValue

	ideSvc, err := o.getIDEService(gitspaceConfig)
	if err != nil {
		return *gitspaceInstance, err
	}

	idePort := ideSvc.Port()

	err = o.infraProvisioner.ResumeProvision(ctx, gitspaceConfig, provisionedInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return *gitspaceInstance, fmt.Errorf(
			"cannot provision infrastructure for ID %s: %w", gitspaceConfig.InfraProviderResource.UID, err)
	}

	if provisionedInfra.Status != enum.InfraStatusProvisioned {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningFailed)

		return *gitspaceInstance, fmt.Errorf(
			"infra state is %v, should be %v for gitspace instance identifier %s",
			provisionedInfra.Status,
			enum.InfraStatusProvisioned,
			gitspaceConfig.GitspaceInstance.Identifier,
		)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeInfraProvisioningCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerStart)

	scmResolvedDetails, err := o.scm.GetSCMRepoDetails(ctx, gitspaceConfig)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerFailed)

		return *gitspaceInstance, fmt.Errorf(
			"failed to fetch code repo details for gitspace config ID %w %d", err, gitspaceConfig.ID)
	}
	devcontainerConfig := scmResolvedDetails.DevcontainerConfig

	if devcontainerConfig == nil {
		log.Warn().Err(err).Msg("devcontainer config is nil, using empty config")
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeFetchDevcontainerCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectStart)

	err = o.containerOrchestrator.Status(ctx, provisionedInfra)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectFailed)

		return *gitspaceInstance, fmt.Errorf("couldn't call the agent health API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentConnectCompleted)

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationStart)

	// NOTE: Currently we use a static identifier as the Gitspace user.
	gitspaceConfig.GitspaceUser.Identifier = harnessUser

	startResponse, err := o.containerOrchestrator.CreateAndStartGitspace(
		ctx, gitspaceConfig, provisionedInfra, *scmResolvedDetails, o.config.DefaultBaseImage, ideSvc)
	if err != nil {
		o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationFailed)

		return *gitspaceInstance, fmt.Errorf("couldn't call the agent start API: %w", err)
	}

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeAgentGitspaceCreationCompleted)

	var ideURL url.URL

	var forwardedPort string

	if provisionedInfra.GitspacePortMappings[idePort.Port].PublishedPort == 0 {
		forwardedPort = startResponse.PublishedPorts[idePort.Port]
	} else {
		forwardedPort = strconv.Itoa(provisionedInfra.GitspacePortMappings[idePort.Port].ForwardedPort)
	}

	scheme := provisionedInfra.GitspaceScheme
	host := provisionedInfra.GitspaceHost
	if provisionedInfra.ProxyGitspaceHost != "" {
		host = provisionedInfra.ProxyGitspaceHost
	}

	relativeRepoPath := strings.TrimPrefix(startResponse.AbsoluteRepoPath, "/")

	if gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
		ideURL = url.URL{
			Scheme:   scheme,
			Host:     host + ":" + forwardedPort,
			RawQuery: filepath.Join("folder=", relativeRepoPath),
		}
	} else if gitspaceConfig.IDE == enum.IDETypeVSCode {
		// TODO: the following userID is hard coded and should be changed.
		ideURL = url.URL{
			Scheme: "vscode-remote",
			Host:   "", // Empty since we include the host and port in the path
			Path: fmt.Sprintf(
				"ssh-remote+%s@%s:%s",
				gitspaceConfig.GitspaceUser.Identifier,
				host,
				filepath.Join(forwardedPort, relativeRepoPath),
			),
		}
	}
	ideURLString := ideURL.String()
	gitspaceInstance.URL = &ideURLString

	now := time.Now().UnixMilli()
	gitspaceInstance.LastUsed = &now
	gitspaceInstance.ActiveTimeStarted = &now
	gitspaceInstance.State = enum.GitspaceInstanceStateRunning

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStartCompleted)

	return *gitspaceInstance, nil
}

func (o orchestrator) getSecretResolver(accessType enum.GitspaceAccessType) (secret.Resolver, error) {
	secretType := secretenum.PasswordSecretType
	switch accessType {
	case enum.GitspaceAccessTypeUserCredentials:
		secretType = secretenum.PasswordSecretType
	case enum.GitspaceAccessTypeJWTToken:
		secretType = secretenum.JWTSecretType
	case enum.GitspaceAccessTypeSSHKey:
		secretType = secretenum.SSHSecretType
	}
	return o.secretResolverFactory.GetSecretResolver(secretType)
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
			"cannot stop provisioned infrastructure with ID %s: %w", gitspaceConfig.InfraProviderResource.UID, err)
	}

	if stoppedInfra.Status != enum.InfraStatusDestroyed &&
		stoppedInfra.Status != enum.InfraStatusStopped {
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

	o.emitGitspaceEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStopCompleted)
	return instanceState, nil
}

func (o orchestrator) getPortsRequiredForGitspace(
	gitspaceConfig types.GitspaceConfig,
) ([]types.GitspacePort, error) {
	// TODO: What if the required ports in the config have deviated from when the last instance was created?
	ideSvc, err := o.getIDEService(gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to get IDE service while checking required Gitspace ports: %w", err)
	}

	idePort := ideSvc.Port()
	return []types.GitspacePort{*idePort}, nil
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
	requiredGitspacePorts, err := o.getPortsRequiredForGitspace(gitspaceConfig)
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
