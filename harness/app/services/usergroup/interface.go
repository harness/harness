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

	"github.com/harness/gitness/types"
)

type Service interface {
	List(
		ctx context.Context,
		filter *types.ListQueryFilter,
		space *types.SpaceCore,
	) ([]*types.UserGroupInfo, error)

	ListUserIDsByGroupIDs(
		ctx context.Context,
		userGroupIDs []int64,
	) ([]int64, error)

	MapGroupIDsToPrincipals(
		ctx context.Context,
		groupIDs []int64,
	) (map[int64][]*types.Principal, error)
}
