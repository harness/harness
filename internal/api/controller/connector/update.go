// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package connector

import (
	"context"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a connector.
type UpdateInput struct {
	UID         *string `json:"uid"`
	Description *string `json:"description"`
	Data        *string `json:"data"`
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
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckConnector(ctx, c.authorizer, session, space.Path, uid, enum.PermissionConnectorEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	connector, err := c.connectorStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to find connector: %w", err)
	}

	return c.connectorStore.UpdateOptLock(ctx, connector, func(original *types.Connector) error {
		if in.UID != nil {
			original.UID = *in.UID
		}
		if in.Description != nil {
			original.Description = *in.Description
		}
		if in.Data != nil {
			original.Data = *in.Data
		}

		return nil
	})
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	if in.UID != nil {
		if err := c.uidCheck(*in.UID, false); err != nil {
			return err
		}
	}

	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	// TODO: Validate Data

	return nil
}
