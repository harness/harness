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

package authz

import (
	"context"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (a *MembershipAuthorizer) CheckPublicAccess(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
	permission enum.Permission,
) (bool, error) {
	// public access only permits view.
	if permission != enum.PermissionRepoView &&
		permission != enum.PermissionSpaceView {
		return false, nil
	}

	// public access is enabled on these resource types.
	if resource.Type != enum.ResourceTypeRepo &&
		resource.Type != enum.ResourceTypeSpace {
		return false, nil
	}

	return a.publicAccess.Get(ctx, scope, resource)
}
