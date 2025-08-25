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

	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Find finds the provisioned infra resources for the gitspace instance.
func (i InfraProvisioner) Find(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (*types.Infrastructure, error) {
	infraProviderResource := gitspaceConfig.InfraProviderResource
	infraProviderEntity, err := i.getConfigFromResource(ctx, infraProviderResource)
	if err != nil {
		return nil, err
	}

	infraProvider, err := i.providerFactory.GetInfraProvider(infraProviderEntity.Type)
	if err != nil {
		return nil, fmt.Errorf("unable to get infra provider of type %v: %w", infraProviderEntity.Type, err)
	}

	var infra *types.Infrastructure
	//nolint:nestif
	if infraProvider.ProvisioningType() == enum.InfraProvisioningTypeNew {
		inputParams, _, err := i.paramsForProvisioningTypeNew(ctx, gitspaceConfig)
		if err != nil {
			return nil, err
		}
		infra, err = i.GetInfraFromStoredInfo(ctx, gitspaceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to find infrastructure from stored info: %w", err)
		}
		status, err := infraProvider.FindInfraStatus(
			ctx,
			gitspaceConfig.Identifier,
			gitspaceConfig.GitspaceInstance.Identifier,
			inputParams,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find infra status: %w", err)
		}
		if status != nil {
			infra.Status = *status
		}
	} else {
		inputParams, _, err := i.paramsForProvisioningTypeExisting(ctx, infraProviderResource, infraProvider)
		if err != nil {
			return nil, err
		}
		infra, err = infraProvider.Find(
			ctx,
			gitspaceConfig.SpaceID,
			gitspaceConfig.SpacePath,
			gitspaceConfig.Identifier,
			inputParams,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find infrastructure from provider: %w", err)
		}
	}

	gitspaceScheme, err := getGitspaceScheme(gitspaceConfig.IDE, infraProviderResource.Metadata["gitspace_scheme"])
	if err != nil {
		return nil, fmt.Errorf("failed to get gitspace_scheme: %w", err)
	}
	infra.GitspaceScheme = gitspaceScheme
	return infra, nil
}

func (i InfraProvisioner) paramsForProvisioningTypeNew(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) ([]types.InfraProviderParameter, map[string]any, error) {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(ctx,
		gitspaceConfig.GitspaceInstance.ID)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w",
			gitspaceConfig.GitspaceInstance.ID, err)
	}
	if infraProvisionedLatest.InputParams == "" {
		return []types.InfraProviderParameter{}, nil, err
	}
	allParams, err := deserializeInfraProviderParams(infraProvisionedLatest.InputParams)
	if err != nil {
		return nil, nil, err
	}
	infraProviderConfig, err := i.infraProviderConfigStore.Find(ctx,
		gitspaceConfig.InfraProviderResource.InfraProviderConfigID, true)
	if err != nil {
		return nil, nil, err
	}
	return allParams, infraProviderConfig.Metadata, nil
}

func deserializeInfraProviderParams(in string) ([]types.InfraProviderParameter, error) {
	var parameters []types.InfraProviderParameter
	err := json.Unmarshal([]byte(in), &parameters)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal infra provider params %+v: %w", in, err)
	}
	return parameters, nil
}

func (i InfraProvisioner) paramsForProvisioningTypeExisting(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
	infraProvider infraprovider.InfraProvider,
) ([]types.InfraProviderParameter, map[string]any, error) {
	allParams, configMetadata, err := i.getAllParamsFromDB(ctx, infraProviderResource, infraProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get all params from DB while finding: %w", err)
	}

	return allParams, configMetadata, nil
}

func getGitspaceScheme(ideType enum.IDEType, gitspaceSchemeFromMetadata string) (string, error) {
	switch ideType {
	case enum.IDETypeVSCodeWeb:
		return gitspaceSchemeFromMetadata, nil
	case enum.IDETypeVSCode, enum.IDETypeWindsurf, enum.IDETypeCursor, enum.IDETypeSSH:
		return "ssh", nil
	case enum.IDETypeIntelliJ, enum.IDETypePyCharm, enum.IDETypeGoland, enum.IDETypeWebStorm, enum.IDETypeCLion,
		enum.IDETypePHPStorm, enum.IDETypeRubyMine, enum.IDETypeRider:
		return "ssh", nil
	default:
		return "", fmt.Errorf("unknown ideType %s", ideType)
	}
}

func (i InfraProvisioner) GetInfraFromStoredInfo(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (*types.Infrastructure, error) {
	infraProvisioned, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx,
		gitspaceConfig.GitspaceInstance.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraProvisioned: %w", err)
	}
	var infra types.Infrastructure
	err = json.Unmarshal([]byte(*infraProvisioned.ResponseMetadata), &infra)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response metadata: %w", err)
	}
	return &infra, nil
}

func (i InfraProvisioner) GetStoppedInfraFromStoredInfo(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (types.Infrastructure, error) {
	var infra types.Infrastructure
	var infraProvisioned *types.InfraProvisioned
	var err error
	if gitspaceConfig.IsMarkedForInfraReset {
		infraProvisioned, err = i.infraProvisionedStore.FindStoppedInfraForGitspaceConfigIdentifierByState(
			ctx,
			gitspaceConfig.Identifier,
			enum.GitspaceInstanceStatePendingCleanup,
		)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to find infraProvisioned with state pending cleanup: %v", err)
		}
	}
	if infraProvisioned == nil {
		infraProvisioned, err = i.infraProvisionedStore.FindStoppedInfraForGitspaceConfigIdentifierByState(
			ctx,
			gitspaceConfig.Identifier,
			enum.GitspaceInstanceStateStarting,
		)
		if err != nil {
			return infra, fmt.Errorf("failed to find infraProvisioned with state starting: %w", err)
		}
	}
	err = json.Unmarshal([]byte(*infraProvisioned.ResponseMetadata), &infra)
	if err != nil {
		// empty infra is returned to avoid nil pointer dereference
		return infra, fmt.Errorf("failed to unmarshal response metadata: %w", err)
	}
	return infra, nil
}

// Methods to find infra provider resources.
func (i InfraProvisioner) getConfigFromResource(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
) (*types.InfraProviderConfig, error) {
	config, err := i.infraProviderConfigStore.Find(ctx, infraProviderResource.InfraProviderConfigID, true)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to get infra provider details for ID %d: %w",
			infraProviderResource.InfraProviderConfigID, err)
	}
	return config, nil
}

func (i InfraProvisioner) getAllParamsFromDB(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
	infraProvider infraprovider.InfraProvider,
) ([]types.InfraProviderParameter, map[string]any, error) {
	var allParams []types.InfraProviderParameter

	templateParams, err := i.getTemplateParams(ctx, infraProvider, infraProviderResource)
	if err != nil {
		return nil, nil, err
	}

	allParams = append(allParams, templateParams...)

	params := i.paramsFromResource(infraProviderResource, infraProvider)

	allParams = append(allParams, params...)

	configMetadata, err := i.configMetadata(ctx, infraProviderResource)
	if err != nil {
		return nil, nil, err
	}

	return allParams, configMetadata, nil
}

func (i InfraProvisioner) getTemplateParams(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	infraProviderResource types.InfraProviderResource,
) ([]types.InfraProviderParameter, error) {
	var params []types.InfraProviderParameter
	templateParams := infraProvider.TemplateParams()

	for _, param := range templateParams {
		key := param.Name
		if infraProviderResource.Metadata[key] != "" {
			templateIdentifier := infraProviderResource.Metadata[key]

			template, err := i.infraProviderTemplateStore.FindByIdentifier(
				ctx, infraProviderResource.SpaceID, templateIdentifier)
			if err != nil {
				return nil, fmt.Errorf("unable to get template params for ID %s: %w",
					infraProviderResource.Metadata[key], err)
			}

			params = append(params, types.InfraProviderParameter{
				Name:  key,
				Value: template.Data,
			})
		}
	}

	return params, nil
}

func (i InfraProvisioner) paramsFromResource(
	infraProviderResource types.InfraProviderResource,
	infraProvider infraprovider.InfraProvider,
) []types.InfraProviderParameter {
	// NOTE: templateParamsMap is required to filter out template params since their values have already been fetched
	// and we dont need the template identifiers, which are the values for template params in the resource Metadata.
	templateParamsMap := make(map[string]bool)
	for _, templateParam := range infraProvider.TemplateParams() {
		templateParamsMap[templateParam.Name] = true
	}

	params := make([]types.InfraProviderParameter, 0, len(infraProviderResource.Metadata))

	for key, value := range infraProviderResource.Metadata {
		if key == "" || value == "" || templateParamsMap[key] {
			continue
		}
		params = append(params, types.InfraProviderParameter{
			Name:  key,
			Value: value,
		})
	}
	return params
}

func (i InfraProvisioner) configMetadata(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
) (map[string]any, error) {
	infraProviderConfig, err := i.infraProviderConfigStore.Find(ctx, infraProviderResource.InfraProviderConfigID, true)
	if err != nil {
		return nil, err
	}
	return infraProviderConfig.Metadata, nil
}
