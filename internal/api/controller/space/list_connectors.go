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

// ListSecrets lists the  connectors in a space.
func (c *Controller) ListConnectors(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.ListQueryFilter,
) ([]*types.Connector, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find parent space: %w", err)
	}

	err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSecretView, false)
	if err != nil {
		return nil, 0, fmt.Errorf("could not authorize: %w", err)
	}

	var count int64
	var connectors []*types.Connector

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.connectorStore.Count(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}

		connectors, err = c.connectorStore.List(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list child executions: %w", err)
		}
		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return connectors, count, fmt.Errorf("failed to list  connectors: %w", err)
	}

	return connectors, count, nil
}
