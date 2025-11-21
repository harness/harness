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

package user

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MembershipSpaces lists all spaces in which the user is a member.
func (c *Controller) MembershipSpaces(ctx context.Context,
	session *auth.Session,
	userUID string,
	filter types.MembershipSpaceFilter,
) ([]types.MembershipSpace, int64, error) {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find user by UID: %w", err)
	}

	// Ensure principal has required permissions.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserView); err != nil {
		return nil, 0, err
	}

	var membershipSpaces []types.MembershipSpace
	var membershipsCount int64

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		membershipSpaces, err = c.membershipStore.ListSpaces(ctx, user.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list membership spaces for user: %w", err)
		}

		if filter.Page == 1 && len(membershipSpaces) < filter.Size {
			membershipsCount = int64(len(membershipSpaces))
			return nil
		}

		membershipsCount, err = c.membershipStore.CountSpaces(ctx, user.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count memberships for user: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return membershipSpaces, membershipsCount, nil
}
