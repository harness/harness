// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
