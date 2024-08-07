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
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ InfraProvisioner = (*infraProvisioner)(nil)

const paramSeparator = "\n===============\n"

type Config struct {
	AgentPort int
}

type infraProvisioner struct {
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
	return &infraProvisioner{
		infraProviderConfigStore:   infraProviderConfigStore,
		infraProviderResourceStore: infraProviderResourceStore,
		providerFactory:            providerFactory,
		infraProviderTemplateStore: infraProviderTemplateStore,
		infraProvisionedStore:      infraProvisionedStore,
		config:                     config,
	}
}

func (i infraProvisioner) getConfigFromResource(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
) (*types.InfraProviderConfig, error) {
	config, err := i.infraProviderConfigStore.Find(ctx, infraProviderResource.InfraProviderConfigID)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to get infra provider details for ID %d: %w",
			infraProviderResource.InfraProviderConfigID, err)
	}
	return config, nil
}

func (i infraProvisioner) getInfraProvider(
	infraProviderType enum.InfraProviderType,
) (infraprovider.InfraProvider, error) {
	infraProvider, err := i.providerFactory.GetInfraProvider(infraProviderType)
	if err != nil {
		return nil, fmt.Errorf("unable to get infra provider of type %v: %w", infraProviderType, err)
	}
	return infraProvider, nil
}

func (i infraProvisioner) getTemplateParams(
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

func (i infraProvisioner) paramsFromResource(
	infraProviderResource types.InfraProviderResource,
) []types.InfraProviderParameter {
	params := make([]types.InfraProviderParameter, 0)
	for key, value := range infraProviderResource.Metadata {
		if key == "" || value == "" {
			continue
		}
		params = append(params, types.InfraProviderParameter{
			Name:  key,
			Value: value,
		})
	}
	return params
}

func paramsToString(in []types.InfraProviderParameter) string {
	var output = ""
	for _, value := range in {
		if value.Name == "" || value.Value == "" {
			continue
		}
		output += value.Name + "=" + value.Value + paramSeparator
	}
	return output
}

func stringToParams(in string) []types.InfraProviderParameter {
	var output = make([]types.InfraProviderParameter, 0)
	paramsRaw := strings.Split(in, paramSeparator)
	for _, value := range paramsRaw {
		parts := strings.Split(value, "=")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		output = append(output, types.InfraProviderParameter{
			Name:  parts[0],
			Value: parts[1],
		})
	}
	return output
}

func (i infraProvisioner) responseMetadata(infra types.Infrastructure) (string, error) {
	output, err := json.Marshal(infra)
	if err != nil {
		return "", fmt.Errorf("unable to marshal infra object %+v: %w", infra, err)
	}

	return string(output), nil
}

func (i infraProvisioner) getAllParamsFromDB(
	ctx context.Context,
	infraProviderResource types.InfraProviderResource,
	infraProvider infraprovider.InfraProvider,
) ([]types.InfraProviderParameter, error) {
	var allParams []types.InfraProviderParameter

	templateParams, err := i.getTemplateParams(ctx, infraProvider, infraProviderResource)
	if err != nil {
		return nil, err
	}

	allParams = append(allParams, templateParams...)

	params := i.paramsFromResource(infraProviderResource)

	allParams = append(allParams, params...)

	return allParams, nil
}

func (i infraProvisioner) updateInfraProvisionedRecord(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	deprovisionedInfra types.Infrastructure,
) error {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(
		ctx, gitspaceConfig.SpaceID, gitspaceConfig.GitspaceInstance.ID)
	if err != nil {
		return fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w",
			gitspaceConfig.GitspaceInstance.ID, err)
	}

	responseMetadata, err := i.responseMetadata(deprovisionedInfra)
	if err != nil {
		return err
	}

	infraProvisionedLatest.InfraStatus = deprovisionedInfra.Status
	infraProvisionedLatest.ServerHostIP = deprovisionedInfra.Host
	infraProvisionedLatest.ServerHostPort = strconv.Itoa(deprovisionedInfra.AgentPort)
	infraProvisionedLatest.ProxyHost = deprovisionedInfra.ProxyHost
	infraProvisionedLatest.ProxyPort = int32(deprovisionedInfra.ProxyPort)
	infraProvisionedLatest.ResponseMetadata = &responseMetadata
	infraProvisionedLatest.Updated = time.Now().UnixMilli()

	err = i.infraProvisionedStore.Update(ctx, infraProvisionedLatest)
	if err != nil {
		return fmt.Errorf("unable to update infraProvisioned %d: %w", infraProvisionedLatest.ID, err)
	}
	return nil
}
