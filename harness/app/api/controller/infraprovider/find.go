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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
)

func (c *Controller) Find(
	ctx context.Context,
	_ *auth.Session,
	spaceRef string,
	identifier string,
) (*types.InfraProviderConfig, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}
	// todo: add acl check with PermissionInfraProviderView once infra provider resource is added to access control
	// err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionGitspaceView)
	// if err != nil {
	//	return nil, fmt.Errorf("failed to authorize: %w", err)
	// }
	return c.infraproviderSvc.Find(ctx, space, identifier)
}
