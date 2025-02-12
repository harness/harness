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

package space

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types/enum"
)

type UpdatePublicAccessInput struct {
	IsPublic bool `json:"is_public"`
}

func (c *Controller) UpdatePublicAccess(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *UpdatePublicAccessInput,
) (*SpaceOutput, error) {
	spaceCore, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	space, err := c.spaceStore.Find(ctx, spaceCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by ID: %w", err)
	}

	parentPath, _, err := paths.DisectLeaf(space.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to disect path %q: %w", space.Path, err)
	}

	isPublicAccessSupported, err := c.publicAccess.IsPublicAccessSupported(ctx, parentPath)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to check if public access is supported for parent space %q: %w",
			parentPath,
			err,
		)
	}
	if in.IsPublic && !isPublicAccessSupported {
		return nil, errPublicSpaceCreationDisabled
	}

	isPublic, err := c.publicAccess.Get(ctx, enum.PublicResourceTypeSpace, space.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check current public access status: %w", err)
	}

	// no op
	if isPublic == in.IsPublic {
		return &SpaceOutput{
			Space:    *space,
			IsPublic: isPublic,
		}, nil
	}

	if err = c.publicAccess.Set(ctx, enum.PublicResourceTypeSpace, space.Path, in.IsPublic); err != nil {
		return nil, fmt.Errorf("failed to update space public access: %w", err)
	}

	return &SpaceOutput{
		Space:    *space,
		IsPublic: in.IsPublic,
	}, nil
}
