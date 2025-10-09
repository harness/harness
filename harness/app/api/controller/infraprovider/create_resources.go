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
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) CreateTemplate(
	ctx context.Context,
	session *auth.Session,
	in *TemplateInput,
	configIdentifier string,
	spaceRef string,
) (*types.InfraProviderTemplate, error) {
	now := time.Now().UnixMilli()
	parentSpace, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}
	if err = apiauth.CheckInfraProvider(
		ctx,
		c.authorizer,
		session,
		parentSpace.Path,
		NoResourceIdentifier,
		enum.PermissionInfraProviderEdit,
	); err != nil {
		return nil, err
	}

	infraProviderConfig, err := c.infraproviderSvc.Find(ctx, parentSpace, configIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider config by ref: %w", err)
	}
	providerTemplate := &types.InfraProviderTemplate{
		Identifier:                    in.Identifier,
		InfraProviderConfigIdentifier: infraProviderConfig.Identifier,
		InfraProviderConfigID:         infraProviderConfig.ID,
		Description:                   in.Description,
		Data:                          in.Data,
		Version:                       0,
		SpaceID:                       parentSpace.ID,
		SpacePath:                     parentSpace.Path,
		Created:                       now,
		Updated:                       now,
	}
	err = c.infraproviderSvc.CreateTemplate(ctx, providerTemplate)
	if err != nil {
		return nil, err
	}
	return providerTemplate, nil
}

func (c *Controller) CreateResources(
	ctx context.Context,
	session auth.Session,
	in []ResourceInput,
	configIdentifier string,
	spaceRef string,
) ([]types.InfraProviderResource, error) {
	if err := c.sanitizeResourceInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	now := time.Now().UnixMilli()
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}
	if err = apiauth.CheckInfraProvider(
		ctx,
		c.authorizer,
		&session,
		space.Path,
		NoResourceIdentifier,
		enum.PermissionInfraProviderEdit,
	); err != nil {
		return nil, err
	}

	resources := c.MapToResourceEntity(in, space, now)
	err = c.infraproviderSvc.CreateResources(ctx, space.ID, resources, configIdentifier)
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (c *Controller) MapToResourceEntity(
	in []ResourceInput,
	space *types.SpaceCore,
	now int64,
) []types.InfraProviderResource {
	var resources []types.InfraProviderResource
	for _, res := range in {
		infraProviderResource := types.InfraProviderResource{
			UID:               res.Identifier,
			InfraProviderType: res.InfraProviderType,
			Name:              res.Name,
			SpaceID:           space.ID,
			CPU:               res.CPU,
			Memory:            res.Memory,
			Disk:              res.Disk,
			Network:           res.Network,
			Region:            res.Region,
			Metadata:          res.Metadata,
			Created:           now,
			Updated:           now,
			SpacePath:         space.Path,
		}
		resources = append(resources, infraProviderResource)
	}
	return resources
}

func (c *Controller) sanitizeResourceInput(in []ResourceInput) error {
	for _, resource := range in {
		if err := check.Identifier(resource.Identifier); err != nil {
			return err
		}
	}
	return nil
}
