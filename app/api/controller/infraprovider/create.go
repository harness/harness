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

const NoResourceIdentifier = ""

type CreateInput struct {
	Identifier string                 `json:"identifier"`
	SpaceRef   string                 `json:"space_ref"` // Ref of the parent space
	Name       string                 `json:"name"`
	Type       enum.InfraProviderType `json:"type"`
	Metadata   map[string]string      `json:"metadata"`
	Resources  []ResourceInput        `json:"resources"`
}

type ResourceInput struct {
	Identifier         string                 `json:"identifier"`
	Name               string                 `json:"name"`
	InfraProviderType  enum.InfraProviderType `json:"infra_provider_type"`
	CPU                *string                `json:"cpu"`
	Memory             *string                `json:"memory"`
	Disk               *string                `json:"disk"`
	Network            *string                `json:"network"`
	Region             []string               `json:"region"`
	Metadata           map[string]string      `json:"metadata"`
	GatewayHost        *string                `json:"gateway_host"`
	GatewayPort        *string                `json:"gateway_port"`
	TemplateIdentifier *string                `json:"template_identifier"`
}

type TemplateInput struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
	Data        string `json:"data"`
}

// Create creates a new infra provider.
func (c *Controller) Create(
	ctx context.Context,
	session auth.Session,
	in CreateInput,
) (*types.InfraProviderConfig, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	parentSpace, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref %q : %w", in.SpaceRef, err)
	}
	if err = apiauth.CheckInfraProvider(
		ctx,
		c.authorizer,
		&session,
		parentSpace.Path,
		NoResourceIdentifier,
		enum.PermissionInfraProviderEdit); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	infraProviderConfig := &types.InfraProviderConfig{
		Identifier: in.Identifier,
		Name:       in.Name,
		SpaceID:    parentSpace.ID,
		Type:       in.Type,
		Created:    now,
		Updated:    now,
	}
	infraProviderConfig.Resources = mapToResourceEntity(in.Resources, *parentSpace, infraProviderConfig.ID)
	err = c.infraproviderSvc.CreateInfraProvider(ctx, infraProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create the infraprovider: %q %w", infraProviderConfig.Identifier, err)
	}
	return infraProviderConfig, nil
}

func (c *Controller) sanitizeCreateInput(in CreateInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	return nil
}
