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

package connector

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Controller) Test(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
) (types.ConnectorTestResponse, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return types.ConnectorTestResponse{}, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckConnector(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionConnectorAccess)
	if err != nil {
		return types.ConnectorTestResponse{}, fmt.Errorf("failed to authorize: %w", err)
	}

	connector, err := c.connectorStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return types.ConnectorTestResponse{}, fmt.Errorf("failed to find connector: %w", err)
	}

	resp, err := c.connectorService.Test(ctx, connector)
	if err != nil {
		return types.ConnectorTestResponse{}, err
	}
	// Try to update connector last test information in DB. Log but ignore errors
	_, err = c.connectorStore.UpdateOptLock(ctx, connector, func(original *types.Connector) error {
		original.LastTestErrorMsg = resp.ErrorMsg
		original.LastTestStatus = resp.Status
		original.LastTestAttempt = time.Now().UnixMilli()
		return nil
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to update test connection information in connector")
	}

	return resp, nil
}
