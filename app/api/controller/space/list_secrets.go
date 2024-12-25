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

// ListSecrets lists the secrets in a space.
func (c *Controller) ListSecrets(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.ListQueryFilter,
) ([]*types.Secret, int64, error) {
	space, err := c.spaceCache.Get(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find parent space: %w", err)
	}

	err = apiauth.CheckSecret(
		ctx,
		c.authorizer,
		session,
		space.Path,
		"",
		enum.PermissionSecretView,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("could not authorize: %w", err)
	}

	var count int64
	var secrets []*types.Secret

	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.secretStore.Count(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}

		secrets, err = c.secretStore.List(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list child executions: %w", err)
		}
		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return secrets, count, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secrets, count, nil
}
