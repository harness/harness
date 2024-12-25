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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListConnectors lists the  connectors in a space.
func (c *Controller) ListConnectors(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.ListQueryFilter,
) ([]*types.Connector, int64, error) {
	space, err := c.spaceCache.Get(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find parent space: %w", err)
	}

	err = apiauth.CheckConnector(
		ctx,
		c.authorizer,
		session,
		space.Path,
		"",
		enum.PermissionConnectorView,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("could not authorize: %w", err)
	}

	count, err := c.connectorStore.Count(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count connectors in space: %w", err)
	}

	connectors, err := c.connectorStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list connectors: %w", err)
	}

	return connectors, count, nil
}
