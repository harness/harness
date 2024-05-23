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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

type MembershipAddInput struct {
	UserUID string              `json:"user_uid"`
	Role    enum.MembershipRole `json:"role"`
}

func (in *MembershipAddInput) Validate() error {
	if in.UserUID == "" {
		return usererror.BadRequest("UserUID must be provided")
	}

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

// MembershipAdd adds a new membership to a space.
func (c *Controller) MembershipAdd(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *MembershipAddInput,
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

	user, err := c.principalStore.FindUserByUID(ctx, in.UserUID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return nil, usererror.BadRequestf("User '%s' not found", in.UserUID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to find the user: %w", err)
	}

	now := time.Now().UnixMilli()

	membership := types.Membership{
		MembershipKey: types.MembershipKey{
			SpaceID:     space.ID,
			PrincipalID: user.ID,
		},
		CreatedBy: session.Principal.ID,
		Created:   now,
		Updated:   now,
		Role:      in.Role,
	}

	err = c.membershipStore.Create(ctx, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to create new membership: %w", err)
	}

	result := &types.MembershipUser{
		Membership: membership,
		Principal:  *user.ToPrincipalInfo(),
		AddedBy:    *session.Principal.ToPrincipalInfo(),
	}

	return result, nil
}
