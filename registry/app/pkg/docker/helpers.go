//  Copyright 2023 Harness, Inc.
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

package docker

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// GetRegistryCheckAccess fetches an active registry
// and checks if the current user has permission to access it.
func GetRegistryCheckAccess(
	ctx context.Context,
	registryStore store.RegistryRepository,
	authorizer authz.Authorizer,
	spaceStore corestore.SpaceStore,
	repoName string,
	parentID int64,
	reqPermissions ...enum.Permission,
) error {
	registry, err := registryStore.GetByParentIDAndName(ctx, parentID, repoName)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	space, err := spaceStore.Find(ctx, parentID)
	if err != nil {
		return fmt.Errorf("failed to find parent by ref: %w", err)
	}
	session, _ := request.AuthSessionFrom(ctx)
	var permissionChecks []types.PermissionCheck

	for i := range reqPermissions {
		permissionCheck := types.PermissionCheck{
			Permission: reqPermissions[i],
			Scope:      types.Scope{SpacePath: space.Identifier},
			Resource: types.Resource{
				Type:       enum.ResourceTypeRegistry,
				Identifier: registry.Name,
			},
		}
		permissionChecks = append(permissionChecks, permissionCheck)
	}

	if err = apiauth.CheckRegistry(ctx, authorizer, session, permissionChecks...); err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}

	return nil
}
