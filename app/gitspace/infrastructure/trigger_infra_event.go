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

	"github.com/rs/zerolog/log"
)

type Config struct {
	AgentPort int
}

type InfraEventOpts struct {
	RequiredGitspacePorts []types.GitspacePort
	CanDeleteUserData     bool
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
	opts := InfraEventOpts{CanDeleteUserData: false}
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

	infraProvider, err := i.getInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return err
	}

	_, configMetadata, err := i.getAllParamsFromDB(ctx, gitspaceConfig.InfraProviderResource, infraProvider)
	if err != nil {
		return fmt.Errorf("could not get all params from DB while provisioning: %w", err)
	}

	switch eventType {
	case enum.InfraEventProvision:
		var stoppedInfra *types.Infrastructure
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
				opts.CanDeleteUserData,
				configMetadata,
			)
		}
		return infraProvider.Deprovision(ctx, *infra, opts.CanDeleteUserData, configMetadata)

	case enum.InfraEventCleanup:
		return infraProvider.CleanupInstanceResources(ctx, *infra)

	case enum.InfraEventStop:
		return infraProvider.Stop(ctx, *infra, configMetadata)

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
	stoppedInfra *types.Infrastructure,
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
	allParams, configMetadata, err := i.getAllParamsFromDB(ctx, infraProviderResource, infraProvider)
	if err != nil {
		return fmt.Errorf("could not get all params from DB while provisioning: %w", err)
	}

	err = infraProvider.ValidateParams(allParams)
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
	}
	if stoppedInfra != nil {
		infrastructure.InstanceInfo = stoppedInfra.InstanceInfo
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
		gitspaceConfig.SpaceID,
		gitspaceConfig.SpacePath,
		gitspaceConfig.Identifier,
		gitspaceConfig.GitspaceInstance.Identifier,
		gitspaceConfig.GitspaceInstance.ID,
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
	stoppedInfra *types.Infrastructure,
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
		gitspaceConfig.SpaceID,
		gitspaceConfig.SpacePath,
		gitspaceConfig.Identifier,
		gitspaceConfig.GitspaceInstance.Identifier,
		gitspaceConfig.GitspaceInstance.ID,
		0, // NOTE: Agent port is not required for provisioning type Existing.
		requiredGitspacePorts,
		allParams,
		configMetadata,
		*stoppedInfra,
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
) error {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx, gitspaceConfig.GitspaceInstance.ID)
	if err != nil {
		return fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w",
			gitspaceConfig.GitspaceInstance.ID, err)
	}

	if infraProvisionedLatest.InfraStatus == enum.InfraStatusDestroyed {
		return nil
	}

	err = infraProvider.Deprovision(ctx, infra, canDeleteUserData, configMetadata)
	if err != nil {
		return fmt.Errorf("unable to trigger deprovision infra %+v: %w", infra, err)
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
