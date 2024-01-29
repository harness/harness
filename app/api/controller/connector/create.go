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
	Description string `json:"description"`
	SpaceRef    string `json:"space_ref"` // Ref of the parent space
	// TODO [CODE-1363]: remove after identifier migration.
	UID        string `json:"uid" deprecated:"true"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	Data       string `json:"data"`
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	in *CreateInput,
) (*types.Connector, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}

	err = apiauth.CheckConnector(ctx, c.authorizer, session, parentSpace.Path, in.Identifier, enum.PermissionConnectorEdit)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	connector := &types.Connector{
		Description: in.Description,
		Data:        in.Data,
		Type:        in.Type,
		SpaceID:     parentSpace.ID,
		Identifier:  in.Identifier,
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
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	parentRefAsID, _ := strconv.ParseInt(in.SpaceRef, 10, 64)

	if parentRefAsID <= 0 || len(strings.TrimSpace(in.SpaceRef)) == 0 {
		return errConnectorRequiresParent
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	return check.Description(in.Description)
}
