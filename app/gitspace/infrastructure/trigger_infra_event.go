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

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Config struct {
	AgentPort   int
	UseSSHPiper bool
}

type InfraEventOpts struct {
	RequiredGitspacePorts []types.GitspacePort
	DeleteUserData        bool
}

type InfraProvisioner struct {
	infraProviderConfigStore   store.InfraProviderConfigStore
	infraProviderResourceStore store.InfraProviderResourceStore
	providerFactory            infraprovider.Factory
	infraProviderTemplateStore store.InfraProviderTemplateStore
	infraProvisionedStore      store.InfraProvisionedStore
	config                     *Config
}

func NewInfraProvisionerService(
	infraProviderConfigStore store.InfraProviderConfigStore,
	infraProviderResourceStore store.InfraProviderResourceStore,
	providerFactory infraprovider.Factory,
	infraProviderTemplateStore store.InfraProviderTemplateStore,
	infraProvisionedStore store.InfraProvisionedStore,
	config *Config,
) InfraProvisioner {
	return InfraProvisioner{
		infraProviderConfigStore:   infraProviderConfigStore,
		infraProviderResourceStore: infraProviderResourceStore,
		providerFactory:            providerFactory,
		infraProviderTemplateStore: infraProviderTemplateStore,
		infraProvisionedStore:      infraProvisionedStore,
		config:                     config,
	}
}

// TriggerInfraEvent an interaction with the infrastructure, and once completed emits event in an asynchronous manner.
func (i InfraProvisioner) TriggerInfraEvent(
	ctx context.Context,
	eventType enum.InfraEvent,
	gitspaceConfig types.GitspaceConfig,
	infra *types.Infrastructure,
) error {
	opts := InfraEventOpts{DeleteUserData: false}
	return i.TriggerInfraEventWithOpts(ctx, eventType, gitspaceConfig, infra, opts)
}

// TriggerInfraEventWithOpts triggers the provisionining of infra resources using the
// infraProviderResource with different infra providers.
// Stop event deprovisions those resources which can be stopped without losing the Gitspace data.
// CleanupInstance event cleans up resources exclusive for a gitspace instance
// Deprovision triggers deprovisioning of resources created for a Gitspace.
// canDeleteUserData = true -> deprovision of all resources
// canDeleteUserData = false -> deprovision of all resources except storage associated to user data.
func (i InfraProvisioner) TriggerInfraEventWithOpts(
	ctx context.Context,
	eventType enum.InfraEvent,
	gitspaceConfig types.GitspaceConfig,
	infra *types.Infrastructure,
	opts InfraEventOpts,
) error {
	logger := log.Logger.With().Logger()
	infraProviderEntity, err := i.getConfigFromResource(ctx, gitspaceConfig.InfraProviderResource)
	if err != nil {
		return err
	}

	infraProvider, err := i.providerFactory.GetInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return fmt.Errorf("unable to get infra provider of type %v: %w", infraProviderEntity.Type, err)
	}

	allParams, configMetadata, err := i.getAllParamsFromDB(ctx, gitspaceConfig.InfraProviderResource, infraProvider)
	if err != nil {
		return fmt.Errorf("could not get all params from DB while provisioning: %w", err)
	}

	switch eventType {
	case enum.InfraEventProvision:
		// We usually suspend/hibernate infra so during subsequent provision we can reuse it.
		// provision resume suspended infra(VM) if it was created before else create new infra during provision.
		var stoppedInfra types.Infrastructure
		stoppedInfra, err = i.GetStoppedInfraFromStoredInfo(ctx, gitspaceConfig)
		if err != nil {
			logger.Info().
				Str("error", err.Error()).
				Msgf(
					"could not find stopped infra provisioned entity for instance %s",
					gitspaceConfig.Identifier,
				)
		}

		if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
			return i.provisionNewInfrastructure(
				ctx,
				infraProvider,
				infraProviderEntity.Type,
				gitspaceConfig,
				opts.RequiredGitspacePorts,
				stoppedInfra,
				configMetadata,
				allParams,
			)
		}
		return i.provisionExistingInfrastructure(
			ctx,
			infraProvider,
			gitspaceConfig,
			opts.RequiredGitspacePorts,
			stoppedInfra,
		)

	case enum.InfraEventDeprovision:
		if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
			return i.deprovisionNewInfrastructure(
				ctx,
				infraProvider,
				gitspaceConfig,
				*infra,
				opts.DeleteUserData,
				configMetadata,
				allParams,
			)
		}
		return infraProvider.Deprovision(ctx, *infra, gitspaceConfig, opts.DeleteUserData, configMetadata, allParams)

	case enum.InfraEventCleanup:
		return infraProvider.CleanupInstanceResources(ctx, *infra)

	case enum.InfraEventStop:
		return infraProvider.Stop(ctx, *infra, gitspaceConfig, configMetadata)

	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func (i InfraProvisioner) provisionNewInfrastructure(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	infraProviderType enum.InfraProviderType,
	gitspaceConfig types.GitspaceConfig,
	requiredGitspacePorts []types.GitspacePort,
	stoppedInfra types.Infrastructure,
	configMetadata map[string]any,
	allParams []types.InfraProviderParameter,
) error {
	// Logic for new provisioning...
	infraProvisionedLatest, _ := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx, gitspaceConfig.GitspaceInstance.ID)

	if infraProvisionedLatest != nil &&
		infraProvisionedLatest.InfraStatus == enum.InfraStatusPending &&
		time.Since(time.UnixMilli(infraProvisionedLatest.Updated)).Milliseconds() < (10*60*1000) {
		return fmt.Errorf("there is already infra provisioning in pending state %d", infraProvisionedLatest.ID)
	} else if infraProvisionedLatest != nil {
		infraProvisionedLatest.InfraStatus = enum.InfraStatusUnknown
		err := i.infraProvisionedStore.Update(ctx, infraProvisionedLatest)
		if err != nil {
			return fmt.Errorf("could not update Infra Provisioned entity: %w", err)
		}
	}

	infraProviderResource := gitspaceConfig.InfraProviderResource

	err := infraProvider.ValidateParams(allParams)
	if err != nil {
		return fmt.Errorf("invalid provisioning params %v: %w", infraProviderResource.Metadata, err)
	}

	now := time.Now()
	paramsBytes, err := serializeInfraProviderParams(allParams)
	if err != nil {
		return err
	}
	infrastructure := types.Infrastructure{
		Identifier:                 gitspaceConfig.GitspaceInstance.Identifier,
		SpaceID:                    gitspaceConfig.SpaceID,
		SpacePath:                  gitspaceConfig.SpacePath,
		GitspaceConfigIdentifier:   gitspaceConfig.Identifier,
		GitspaceInstanceIdentifier: gitspaceConfig.GitspaceInstance.Identifier,
		ProviderType:               infraProviderType,
		InputParameters:            allParams,
		ConfigMetadata:             configMetadata,
		InstanceInfo:               stoppedInfra.InstanceInfo,
		Status:                     enum.InfraStatusPending,
	}

	if i.useRoutingKey(infraProviderType, gitspaceConfig.GitspaceInstance.AccessType) {
		infrastructure.RoutingKey = i.getRoutingKey(gitspaceConfig.SpacePath, gitspaceConfig.Identifier)
	}

	responseMetadata, err := json.Marshal(infrastructure)
	if err != nil {
		return fmt.Errorf("unable to marshal infra object %+v: %w", responseMetadata, err)
	}
	responseMetaDataJSON := string(responseMetadata)

	infraProvisioned := &types.InfraProvisioned{
		GitspaceInstanceID:      gitspaceConfig.GitspaceInstance.ID,
		InfraProviderType:       infraProviderType,
		InfraProviderResourceID: infraProviderResource.ID,
		Created:                 now.UnixMilli(),
		Updated:                 now.UnixMilli(),
		InputParams:             paramsBytes,
		InfraStatus:             enum.InfraStatusPending,
		SpaceID:                 gitspaceConfig.SpaceID,
		ResponseMetadata:        &responseMetaDataJSON,
	}

	err = i.infraProvisionedStore.Create(ctx, infraProvisioned)
	if err != nil {
		return fmt.Errorf("unable to create infraProvisioned entry for %d", gitspaceConfig.GitspaceInstance.ID)
	}

	agentPort := i.config.AgentPort

	err = infraProvider.Provision(
		ctx,
		gitspaceConfig,
		agentPort,
		requiredGitspacePorts,
		allParams,
		configMetadata,
		infrastructure,
	)
	if err != nil {
		infraProvisioned.InfraStatus = enum.InfraStatusUnknown
		infraProvisioned.Updated = time.Now().UnixMilli()
		updateErr := i.infraProvisionedStore.Update(ctx, infraProvisioned)
		if updateErr != nil {
			log.Err(updateErr).Msgf("unable to update infraProvisioned Entry for %d", infraProvisioned.ID)
		}

		return fmt.Errorf(
			"unable to trigger provision infrastructure for gitspaceConfigIdentifier %v: %w",
			gitspaceConfig.Identifier,
			err,
		)
	}

	return nil
}

func (i InfraProvisioner) provisionExistingInfrastructure(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	gitspaceConfig types.GitspaceConfig,
	requiredGitspacePorts []types.GitspacePort,
	stoppedInfra types.Infrastructure,
) error {
	allParams, configMetadata, err := i.getAllParamsFromDB(ctx, gitspaceConfig.InfraProviderResource, infraProvider)
	if err != nil {
		return fmt.Errorf("could not get all params from DB while provisioning: %w", err)
	}

	err = infraProvider.ValidateParams(allParams)
	if err != nil {
		return fmt.Errorf("invalid provisioning params %v: %w", gitspaceConfig.InfraProviderResource.Metadata, err)
	}

	err = infraProvider.Provision(
		ctx,
		gitspaceConfig,
		0, // NOTE: Agent port is not required for provisioning type Existing.
		requiredGitspacePorts,
		allParams,
		configMetadata,
		stoppedInfra,
	)
	if err != nil {
		return fmt.Errorf(
			"unable to trigger provision infrastructure for gitspaceConfigIdentifier %v: %w",
			gitspaceConfig.Identifier,
			err,
		)
	}

	return nil
}

func (i InfraProvisioner) deprovisionNewInfrastructure(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	canDeleteUserData bool,
	configMetadata map[string]any,
	params []types.InfraProviderParameter,
) error {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx, gitspaceConfig.GitspaceInstance.ID)
	if err != nil {
		return fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w",
			gitspaceConfig.GitspaceInstance.ID, err)
	}

	if infraProvisionedLatest.InfraStatus == enum.InfraStatusDestroyed {
		log.Debug().Msgf(
			"infra already deprovisioned for gitspace instance %d", gitspaceConfig.GitspaceInstance.ID)
	}

	err = infraProvider.Deprovision(ctx, infra, gitspaceConfig, canDeleteUserData, configMetadata, params)
	if err != nil {
		return fmt.Errorf("unable to trigger deprovision infra %s: %w", infra.Identifier, err)
	}

	return err
}

func serializeInfraProviderParams(in []types.InfraProviderParameter) (string, error) {
	output, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("unable to marshal infra provider params: %w", err)
	}
	return string(output), nil
}

func (i InfraProvisioner) useRoutingKey(
	infraProviderType enum.InfraProviderType,
	accessType enum.GitspaceAccessType,
) bool {
	return i.config.UseSSHPiper && infraProviderType == enum.InfraProviderTypeHybridVMAWS &&
		accessType == enum.GitspaceAccessTypeSSHKey
}

func (i InfraProvisioner) getRoutingKey(spacePath string, gitspaceConfigIdentifier string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL,
		fmt.Appendf(nil, "%s%s", spacePath, gitspaceConfigIdentifier)).String()
}
