// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package connector

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a connector.
type UpdateInput struct {
	Description string `json:"description"`
	UID         string `json:"uid"`
	Data        string `json:"data"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	uid string,
	in *UpdateInput,
) (*types.Connector, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("could not find space: %w", err)
	}

	err = apiauth.CheckConnector(ctx, c.authorizer, session, space.Path, uid, enum.PermissionConnectorEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	connector, err := c.connectorStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("could not find connector: %w", err)
	}

	return c.connectorStore.UpdateOptLock(ctx, connector, func(original *types.Connector) error {
		if in.Description != "" {
			original.Description = in.Description
		}
		if in.Data != "" {
			original.Data = in.Data
		}
		if in.UID != "" {
			original.UID = in.UID
		}

		return nil
	})
}
