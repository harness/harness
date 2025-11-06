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
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type SoftDeleteResponse struct {
	DeletedAt int64 `json:"deleted_at"`
}

// SoftDelete marks deleted timestamp for the space and all its subspaces and repositories inside.
func (c *Controller) SoftDelete(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (*SoftDeleteResponse, error) {
	spaceCore, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceDelete)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	space, err := c.spaceStore.Find(ctx, spaceCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by ID: %w", err)
	}

	return c.SoftDeleteNoAuth(ctx, session, space)
}

// SoftDeleteNoAuth soft deletes the space - no authorization is verified.
// WARNING For internal calls only.
func (c *Controller) SoftDeleteNoAuth(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
) (*SoftDeleteResponse, error) {
	err := c.publicAccess.Delete(ctx, enum.PublicResourceTypeSpace, space.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete public access for space: %w", err)
	}

	var deletedAt int64
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := c.spaceStore.FindForUpdate(ctx, space.ID)
		if err != nil {
			return fmt.Errorf("failed to lock the space for update: %w", err)
		}
		deletedAt = time.Now().UnixMilli()
		err = c.spaceSvc.SoftDeleteInner(ctx, session, space, deletedAt)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to soft delete the space: %w", err)
	}

	c.spaceFinder.MarkChanged(ctx, space.Core())

	return &SoftDeleteResponse{DeletedAt: deletedAt}, nil
}
