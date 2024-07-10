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

package gitspace

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a gitspace.
type UpdateInput struct {
	IDE                enum.IDEType `json:"ide"`
	ResourceIdentifier string       `json:"resource_identifier"`
	Name               string       `json:"name"`
	Identifier         string       `json:"-"`
	SpaceRef           string       `json:"-"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	in *UpdateInput,
) error {
	in.SpaceRef = spaceRef
	in.Identifier = identifier
	if err := c.sanitizeUpdateInput(in); err != nil {
		return fmt.Errorf("failed to sanitize input: %w", err)
	}
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionGitspaceEdit)
	if err != nil {
		return fmt.Errorf("failed to authorize: %w", err)
	}

	gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return fmt.Errorf("failed to find gitspace config: %w", err)
	}
	// TODO Update with proper locks
	return c.gitspaceConfigStore.Update(ctx, gitspaceConfig)
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return ErrGitspaceRequiresParent
	}

	//nolint:revive
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	return nil
}
