// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/pipeline/events"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Events(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (<-chan *events.Event, <-chan error, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find space ref: %w", err)
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, true); err != nil {
		return nil, nil, fmt.Errorf("failed to authorize stream: %w", err)
	}

	events, errc := c.eventsStream.Subscribe(ctx, space.ID)
	return events, errc, nil
}
