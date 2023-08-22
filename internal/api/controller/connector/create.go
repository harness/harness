// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package connector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	// errConnectorRequiresParent if the user tries to create a connector without a parent space.
	errConnectorRequiresParent = usererror.BadRequest(
		"Parent space required - standalone connector are not supported.")
)

type CreateInput struct {
	Description string `json:"description"`
	SpaceRef    string `json:"space_ref"` // Ref of the parent space
	UID         string `json:"uid"`
	Type        string `json:"type"`
	Data        string `json:"data"`
}

func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Connector, error) {
	parentSpace, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}

	err = apiauth.CheckConnector(ctx, c.authorizer, session, parentSpace.Path, in.UID, enum.PermissionConnectorEdit)
	if err != nil {
		return nil, err
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	var connector *types.Connector
	now := time.Now().UnixMilli()
	connector = &types.Connector{
		Description: in.Description,
		Data:        in.Data,
		Type:        in.Type,
		SpaceID:     parentSpace.ID,
		UID:         in.UID,
		Created:     now,
		Updated:     now,
		Version:     0,
	}
	err = c.connectorStore.Create(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("connector creation failed: %w", err)
	}

	return connector, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)

	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return errConnectorRequiresParent
	}

	if err := c.uidCheck(in.UID, false); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	return check.Description(in.Description)

}
