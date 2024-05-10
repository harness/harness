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

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (a *MembershipAuthorizer) CheckPublicAccess(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
	permission enum.Permission,
) (bool, error) {
	var pubResType enum.PublicResourceType

	//nolint:exhaustive
	switch resource.Type {
	case enum.ResourceTypeSpace:
		pubResType = enum.PublicResourceTypeSpace

	case enum.ResourceTypeRepo:
		if resource.Identifier != "" {
			pubResType = enum.PublicResourceTypeRepo
		} else { // for spaceScope checks
			pubResType = enum.PublicResourceTypeSpace
		}

	case enum.ResourceTypePipeline:
		pubResType = enum.PublicResourceTypeRepo

	default:
		return false, nil
	}

	pubResPath := paths.Concatenate(scope.SpacePath, resource.Identifier)

	return a.publicAccess.Get(ctx, pubResType, pubResPath)
}
