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

package principal

import (
	"context"

	"github.com/harness/gitness/types"
)

func (c controller) List(ctx context.Context, opts *types.PrincipalFilter) (
	[]*types.PrincipalInfo, error) {
	principals, err := c.principalStore.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	pInfoUsers := make([]*types.PrincipalInfo, len(principals))
	for i := range principals {
		pInfoUsers[i] = principals[i].ToPrincipalInfo()
	}

	return pInfoUsers, nil
}
