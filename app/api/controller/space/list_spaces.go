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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Space struct {
	types.Space
	IsPublic bool `json:"is_public"`
}

// ListSpaces lists the child spaces of a space.
func (c *Controller) ListSpaces(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter *types.SpaceFilter,
) ([]*Space, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpaceScope(
		ctx,
		c.authorizer,
		session,
		space,
		enum.ResourceTypeSpace,
		enum.PermissionSpaceView,
		true,
	); err != nil {
		return nil, 0, err
	}

	return c.ListSpacesNoAuth(ctx, space.ID, filter)
}

// ListSpacesNoAuth lists spaces WITHOUT checking PermissionSpaceView.
func (c *Controller) ListSpacesNoAuth(
	ctx context.Context,
	spaceID int64,
	filter *types.SpaceFilter,
) ([]*Space, int64, error) {
	var spaces []*Space
	var count int64

	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.spaceStore.Count(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child spaces: %w", err)
		}

		spacesBase, err := c.spaceStore.List(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to list child spaces: %w", err)
		}

		for _, spaceBase := range spacesBase {
			// backfill public access mode
			isPublic, err := c.publicAccess.Get(ctx, &types.PublicResource{
				Type:       enum.PublicResourceTypeSpace,
				ResourceID: spaceBase.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to get resource public access mode: %w", err)
			}

			spaces = append(spaces, &Space{
				Space:    *spaceBase,
				IsPublic: isPublic,
			})
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return spaces, count, nil
}
