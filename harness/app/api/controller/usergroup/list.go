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

package usergroup

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c Controller) List(
	ctx context.Context,
	session *auth.Session,
	filter *types.ListQueryFilter,
	spaceRef string,
) ([]*types.UserGroupInfo, error) {
	space, err := getSpaceCheckAuth(
		ctx, c.spaceFinder, c.authorizer, session, spaceRef, enum.PermissionSpaceView,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	userGroupInfos, err := c.userGroupService.List(ctx, filter, space)
	if err != nil {
		return nil, fmt.Errorf("failed to search user groups: %w", err)
	}
	return userGroupInfos, nil
}
