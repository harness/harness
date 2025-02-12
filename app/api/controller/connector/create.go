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
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
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
	Description string             `json:"description"`
	SpaceRef    string             `json:"space_ref"` // Ref of the parent space
	Identifier  string             `json:"identifier"`
	Type        enum.ConnectorType `json:"type"`
	types.ConnectorConfig
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	in *CreateInput,
) (*types.Connector, error) {
	if err := in.validate(); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.spaceFinder.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}

	err = apiauth.CheckConnector(
		ctx,
		c.authorizer,
		session,
		parentSpace.Path,
		"",
		enum.PermissionConnectorEdit,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	connector := &types.Connector{
		Description:     in.Description,
		CreatedBy:       session.Principal.ID,
		Type:            in.Type,
		SpaceID:         parentSpace.ID,
		Identifier:      in.Identifier,
		Created:         now,
		Updated:         now,
		Version:         0,
		ConnectorConfig: in.ConnectorConfig,
	}

	err = c.connectorStore.Create(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("connector creation failed: %w", err)
	}

	return connector, nil
}

func (in *CreateInput) validate() error {
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return errConnectorRequiresParent
	}

	// check that the connector type is valid
	if _, ok := in.Type.Sanitize(); !ok {
		return usererror.BadRequest("invalid connector type")
	}

	// if the connector type is valid, validate the connector config
	if err := in.ConnectorConfig.Validate(in.Type); err != nil {
		return usererror.BadRequest(fmt.Sprintf("invalid connector config: %s", err.Error()))
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	return check.Description(in.Description)
}
