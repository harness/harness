// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
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
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find parent space: %w", err)
	}

	err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSecretView, false)
	if err != nil {
		return nil, 0, fmt.Errorf("could not authorize: %w", err)
	}

	var count int64
	var secrets []*types.Secret

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) (err error) {
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
