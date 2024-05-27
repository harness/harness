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

func (c *Controller) ListPublicKeys(
	ctx context.Context,
	session *auth.Session,
	userUID string,
	filter *types.PublicKeyFilter,
) ([]types.PublicKey, int, error) {
	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch user by uid: %w", err)
	}

	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserView); err != nil {
		return nil, 0, err
	}

	var (
		list  []types.PublicKey
		count int
	)

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		list, err = c.publicKeyStore.List(ctx, user.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list public keys for user: %w", err)
		}

		if filter.Page == 1 && len(list) < filter.Size {
			count = len(list)
			return nil
		}

		count, err = c.publicKeyStore.Count(ctx, user.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count public keys for user: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return list, count, nil
}
