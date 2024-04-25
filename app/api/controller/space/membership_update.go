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
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type MembershipUpdateInput struct {
	Role enum.MembershipRole `json:"role"`
}

func (in *MembershipUpdateInput) Validate() error {
	if in.Role == "" {
		return usererror.BadRequest("Role must be provided")
	}

	role, ok := in.Role.Sanitize()
	if !ok {
		msg := fmt.Sprintf("Provided role '%s' is not suppored. Valid values are: %v",
			in.Role, enum.MembershipRoles)
		return usererror.BadRequest(msg)
	}

	in.Role = role

	return nil
}

// MembershipUpdate changes the role of an existing membership.
func (c *Controller) MembershipUpdate(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	userUID string,
	in *MembershipUpdateInput,
) (*types.MembershipUser, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit); err != nil {
		return nil, err
	}

	err = in.Validate()
	if err != nil {
		return nil, err
	}

	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by uid: %w", err)
	}

	membership, err := c.membershipStore.FindUser(ctx, types.MembershipKey{
		SpaceID:     space.ID,
		PrincipalID: user.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find membership for update: %w", err)
	}

	if membership.Role == in.Role {
		return membership, nil
	}

	membership.Role = in.Role

	err = c.membershipStore.Update(ctx, &membership.Membership)
	if err != nil {
		return nil, fmt.Errorf("failed to update membership")
	}

	return membership, nil
}
