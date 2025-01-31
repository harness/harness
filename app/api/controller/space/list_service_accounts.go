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

// ListServiceAccounts lists the service accounts of a space.
func (c *Controller) ListServiceAccounts(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	inherited bool,
	opts *types.PrincipalFilter,
) ([]*types.ServiceAccountInfo, int64, error) {
	space, err := c.spaceCache.Get(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckServiceAccount(
		ctx,
		c.authorizer,
		session,
		c.spaceStore,
		c.repoStore,
		enum.ParentResourceTypeSpace,
		space.ID,
		"",
		enum.PermissionServiceAccountView,
	); err != nil {
		return nil, 0, err
	}

	var parentInfos []*types.ServiceAccountParentInfo
	if inherited {
		ancestorIDs, err := c.spaceStore.GetAncestorIDs(ctx, space.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get parent space ids: %w", err)
		}

		parentInfos = make([]*types.ServiceAccountParentInfo, len(ancestorIDs))
		for i := range ancestorIDs {
			parentInfos[i] = &types.ServiceAccountParentInfo{
				Type: enum.ParentResourceTypeSpace,
				ID:   ancestorIDs[i],
			}
		}
	} else {
		parentInfos = make([]*types.ServiceAccountParentInfo, 1)
		parentInfos[0] = &types.ServiceAccountParentInfo{
			Type: enum.ParentResourceTypeSpace,
			ID:   space.ID,
		}
	}

	var accounts []*types.ServiceAccount
	var count int64
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		accounts, err = c.principalStore.ListServiceAccounts(ctx, parentInfos, opts)
		if err != nil {
			return fmt.Errorf("failed to list service accounts: %w", err)
		}

		if opts.Page == 1 && len(accounts) < opts.Size {
			count = int64(len(accounts))
			return nil
		}

		count, err = c.principalStore.CountServiceAccounts(ctx, parentInfos, opts)
		if err != nil {
			return fmt.Errorf("failed to count pull requests: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)

	infos := make([]*types.ServiceAccountInfo, len(accounts))
	for i := range accounts {
		infos[i] = accounts[i].ToServiceAccountInfo()
	}

	return infos, count, nil
}
