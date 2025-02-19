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

// CreateConfig creates a new infra provider config.
func (c *Controller) CreateConfig(
	ctx context.Context,
	session auth.Session,
	in ConfigInput,
) (*types.InfraProviderConfig, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	parentSpace, err := c.spaceFinder.FindByRef(ctx, in.SpaceRef)
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
	infraProviderConfig := c.MapToInfraProviderConfig(in, parentSpace, now)
	err = c.infraproviderSvc.CreateConfig(ctx, infraProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create the infraprovider: %q %w", infraProviderConfig.Identifier, err)
	}
	return infraProviderConfig, nil
}

func (c *Controller) MapToInfraProviderConfig(
	in ConfigInput,
	space *types.SpaceCore,
	now int64,
) *types.InfraProviderConfig {
	return &types.InfraProviderConfig{
		Identifier: in.Identifier,
		Name:       in.Name,
		SpaceID:    space.ID,
		SpacePath:  space.Path,
		Type:       in.Type,
		Created:    now,
		Updated:    now,
		Metadata:   in.Metadata,
	}
}

func (c *Controller) sanitizeCreateInput(in ConfigInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	return nil
}
