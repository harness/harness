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
	"fmt"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CheckPublicAccess checks if the requested permission is public for the provided scope and resource.
func CheckPublicAccess(
	ctx context.Context,
	publicAccess publicaccess.Service,
	scope *types.Scope,
	resource *types.Resource,
	permission enum.Permission,
) (bool, error) {
	var pubResType enum.PublicResourceType
	var pubResPath string

	//nolint:exhaustive
	switch resource.Type {
	case enum.ResourceTypeSpace:
		pubResType = enum.PublicResourceTypeSpace
		pubResPath = paths.Concatenate(scope.SpacePath, resource.Identifier)

	case enum.ResourceTypeRepo:
		if resource.Identifier != "" {
			pubResType = enum.PublicResourceTypeRepo
			pubResPath = paths.Concatenate(scope.SpacePath, resource.Identifier)
		} else { // for spaceScope checks
			pubResType = enum.PublicResourceTypeSpace
			pubResPath = scope.SpacePath
		}

	case enum.ResourceTypePipeline:
		pubResType = enum.PublicResourceTypeRepo
		pubResPath = paths.Concatenate(scope.SpacePath, scope.Repo)

	default:
		return false, nil
	}

	// only view permissions allowed for public access.
	if pubResType == enum.PublicResourceTypeRepo &&
		(permission != enum.PermissionRepoView &&
			permission != enum.PermissionPipelineView) {
		return false, nil
	}

	if pubResType == enum.PublicResourceTypeSpace &&
		permission != enum.PermissionSpaceView {
		return false, nil
	}

	resourceIsPublic, err := publicAccess.Get(ctx, pubResType, pubResPath)
	if err != nil {
		return false, fmt.Errorf("failed to check public accessabillity of %s %q: %w", pubResType, pubResPath, err)
	}

	return resourceIsPublic, nil
}
