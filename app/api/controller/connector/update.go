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
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a connector.
type UpdateInput struct {
	Identifier  *string `json:"identifier"`
	Description *string `json:"description"`
	*types.ConnectorConfig
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	in *UpdateInput,
) (*types.Connector, error) {
	if err := in.validate(); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckConnector(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionConnectorEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	connector, err := c.connectorStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find connector: %w", err)
	}

	return c.connectorStore.UpdateOptLock(ctx, connector, func(original *types.Connector) error {
		if in.Identifier != nil {
			original.Identifier = *in.Identifier
		}
		if in.Description != nil {
			original.Description = *in.Description
		}
		// TODO: See if this can be made better. The PATCH API supports partial updates so
		// currently we keep all the top level fields the same unless they are explicitly provided.
		// The connector config is a nested field so we only check whether it's provided at the top level, and not
		// all the fields inside the config. Maybe PUT/POST would be a better option here?
		// We can revisit this once we start adding more connectors.
		if in.ConnectorConfig != nil {
			if err := in.ConnectorConfig.Validate(connector.Type); err != nil {
				return usererror.BadRequestf("failed to validate connector config: %s", err.Error())
			}
			original.ConnectorConfig = *in.ConnectorConfig
		}
		return nil
	})
}

func (in *UpdateInput) validate() error {
	if in.Identifier != nil {
		if err := check.Identifier(*in.Identifier); err != nil {
			return err
		}
	}

	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	return nil
}
