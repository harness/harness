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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListResources retrieves all resources for an infrastructure provider.
func (c *Controller) ListResources(
	ctx context.Context,
	session *auth.Session,
	spaceRef, infraProviderIdentifier string,
) ([]*types.InfraProviderResource, error) {
	// Find the space for authorization checks
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	// Check authorization
	err = apiauth.CheckInfraProvider(ctx, c.authorizer, session, space.Path, "", enum.PermissionInfraProviderView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	// Find the infra provider config using the correct method
	config, err := c.infraproviderSvc.Find(ctx, space, infraProviderIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infra provider: %w", err)
	}

	// The config from Find() already has its resources populated, so we can just use them
	// Create pointers for the resources from the populated config
	resources := make([]*types.InfraProviderResource, len(config.Resources))
	for i := range config.Resources {
		resources[i] = &config.Resources[i]
	}

	return resources, nil
}
