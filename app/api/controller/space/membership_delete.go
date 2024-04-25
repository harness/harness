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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MembershipDelete removes an existing membership from a space.
func (c *Controller) MembershipDelete(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	userUID string,
) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit); err != nil {
		return err
	}

	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return fmt.Errorf("failed to find user by uid: %w", err)
	}

	err = c.membershipStore.Delete(ctx, types.MembershipKey{
		SpaceID:     space.ID,
		PrincipalID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user membership: %w", err)
	}

	return nil
}
