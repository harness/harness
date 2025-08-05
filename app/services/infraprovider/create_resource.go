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

package infraprovider

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Service) CreateResources(
	ctx context.Context,
	spaceID int64,
	resources []types.InfraProviderResource,
	configIdentifier string,
) error {
	config, err := c.infraProviderConfigStore.FindByIdentifier(ctx, spaceID, configIdentifier)
	if err != nil {
		return fmt.Errorf("failed to find config: %w", err)
	}

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		return c.upsertResources(ctx, resources, config.ID, spaceID, *config, false)
	})
	if err != nil {
		return fmt.Errorf("failed to complete create txn for the infraprovider resource %w", err)
	}
	return nil
}

func (c *Service) upsertResources(
	ctx context.Context,
	resources []types.InfraProviderResource,
	configID int64,
	spaceID int64,
	config types.InfraProviderConfig,
	allowUpdates bool,
) error {
	emptyStr := ""
	for idx := range resources {
		resource := &resources[idx]
		resource.InfraProviderConfigID = configID
		resource.SpaceID = spaceID

		if resource.Network == nil {
			resource.Network = &emptyStr
		}
		// updating metadata based on infra provider type
		updatedMetadata, err := c.updateResourceMetadata(resource, config)
		if err != nil {
			return fmt.Errorf("creating missing infra resources: %w", err)
		}
		resource.Metadata = updatedMetadata

		cpuStr := getMetadataVal(updatedMetadata, "cpu")
		memoryStr := getMetadataVal(updatedMetadata, "memory")
		if resource.CPU == nil || (resource.CPU != nil && *resource.CPU == "") {
			resource.CPU = &cpuStr
		}

		if resource.Memory == nil || (resource.Memory != nil && *resource.Memory == "") {
			resource.Memory = &memoryStr
		}

		if err := c.validateResource(ctx, resource); err != nil {
			return err
		}
		existingResource, err := c.infraProviderResourceStore.FindByConfigAndIdentifier(ctx, resource.SpaceID,
			configID, resource.UID)
		if err != nil {
			if !errors.Is(err, store.ErrResourceNotFound) {
				return fmt.Errorf("failed to check existing resource %s: %w", resource.UID, err)
			}
			// Resource doesn't exist, create it
			if err = c.infraProviderResourceStore.Create(ctx, resource); err != nil {
				return fmt.Errorf("failed to create infraprovider resource for %s: %w", resource.UID, err)
			}
		} else {
			// Resource exists
			if allowUpdates {
				if err := c.updateExistingResource(ctx, resource, existingResource); err != nil {
					return fmt.Errorf("failed to update existing resource %s: %w", resource.UID, err)
				}
				log.Info().Msgf(
					"updated existing resource %s/%s",
					resource.InfraProviderConfigIdentifier,
					resource.UID,
				)
			} else {
				return fmt.Errorf("resource %s already exists", resource.UID)
			}
		}
	}
	return nil
}

func (c *Service) updateResourceMetadata(
	resource *types.InfraProviderResource,
	config types.InfraProviderConfig,
) (map[string]string, error) {
	infraProvider, err := c.infraProviderFactory.GetInfraProvider(resource.InfraProviderType)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch infra impl for type : %q %w", resource.InfraProviderType, err)
	}

	params, err := infraProvider.UpdateParams(toResourceParams(resource.Metadata), config.Metadata)
	if err != nil {
		return nil, err
	}

	return toMetadata(params), nil
}

func (c *Service) validateResource(ctx context.Context, resource *types.InfraProviderResource) error {
	infraProvider, err := c.infraProviderFactory.GetInfraProvider(resource.InfraProviderType)
	if err != nil {
		return fmt.Errorf("failed to fetch infra impl for type : %q %w", resource.InfraProviderType, err)
	}

	if len(infraProvider.TemplateParams()) > 0 {
		err = c.validateTemplates(ctx, infraProvider, *resource)
		if err != nil {
			return err
		}
	}

	if len(resource.Metadata) > 0 && resource.Metadata["resource_name"] == "" {
		resource.Metadata["resource_name"] = resource.Name
	}

	resourceParams := toResourceParams(resource.Metadata)

	err = infraProvider.ValidateParams(resourceParams)
	if err != nil {
		return err
	}

	return err
}

func toResourceParams(metadata map[string]string) []types.InfraProviderParameter {
	var infraResourceParams []types.InfraProviderParameter
	for key, value := range metadata {
		infraResourceParams = append(infraResourceParams, types.InfraProviderParameter{
			Name:  key,
			Value: value,
		})
	}

	return infraResourceParams
}

func toMetadata(params []types.InfraProviderParameter) map[string]string {
	metadata := make(map[string]string)

	for _, param := range params {
		metadata[param.Name] = param.Value
	}

	return metadata
}

func getMetadataVal(metadata map[string]string, key string) string {
	if val, ok := metadata[key]; ok {
		return val
	}
	return ""
}

// updateExistingResource updates an existing resource with new information while preserving
// immutable fields.
func (c *Service) updateExistingResource(
	ctx context.Context,
	resource *types.InfraProviderResource,
	existingResource *types.InfraProviderResource,
) error {
	// Keep the ID and created timestamp from the existing resource
	resource.ID = existingResource.ID
	resource.Created = existingResource.Created

	// Preserve immutable fields
	resource.UID = existingResource.UID
	resource.InfraProviderConfigID = existingResource.InfraProviderConfigID
	resource.SpaceID = existingResource.SpaceID
	resource.InfraProviderType = existingResource.InfraProviderType

	// Update the resource
	return c.infraProviderResourceStore.Update(ctx, resource)
}
