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

package auth

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CheckSpace checks if a space specific permission is granted for the current auth session
// in the scope of its parent.
// Returns nil if permission is granted, otherwise returns NotAuthenticated, NotAuthorized, or the underlying error.
func CheckSpace(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	space *types.Space,
	permission enum.Permission,
) error {
	parentSpace, name, err := paths.DisectLeaf(space.Path)
	if err != nil {
		return fmt.Errorf("failed to disect path '%s': %w", space.Path, err)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type:       enum.ResourceTypeSpace,
		Identifier: name,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}

// CheckSpaceScope checks if a specific permission is granted for the current auth session
// in the scope of the provided space.
// Returns nil if permission is granted, otherwise returns NotAuthenticated, NotAuthorized, or the underlying error.
func CheckSpaceScope(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	space *types.Space,
	resourceType enum.ResourceType,
	permission enum.Permission,
) error {
	scope := &types.Scope{SpacePath: space.Path}
	resource := &types.Resource{
		Type:       resourceType,
		Identifier: "",
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}
