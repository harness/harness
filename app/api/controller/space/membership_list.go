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

// MembershipList lists all space memberships.
func (c *Controller) MembershipList(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.MembershipUserFilter,
) ([]types.MembershipUser, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView); err != nil {
		return nil, 0, err
	}

	var memberships []types.MembershipUser
	var membershipsCount int64

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		memberships, err = c.membershipStore.ListUsers(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list memberships for space: %w", err)
		}

		if filter.Page == 1 && len(memberships) < filter.Size {
			membershipsCount = int64(len(memberships))
			return nil
		}

		membershipsCount, err = c.membershipStore.CountUsers(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count memberships for space: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return memberships, membershipsCount, nil
}
