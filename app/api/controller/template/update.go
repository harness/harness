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

package template

import (
	"context"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a template.
type UpdateInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID         *string `json:"uid" deprecated:"true"`
	Identifier  *string `json:"identifier"`
	Description *string `json:"description"`
	Data        *string `json:"data"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	resolverType enum.ResolverType,
	in *UpdateInput,
) (*types.Template, error) {
	if err := c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckTemplate(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionTemplateEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	template, err := c.templateStore.FindByIdentifierAndType(ctx, space.ID, identifier, resolverType)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	return c.templateStore.UpdateOptLock(ctx, template, func(original *types.Template) error {
		if in.Identifier != nil {
			original.Identifier = *in.Identifier
		}
		if in.Description != nil {
			original.Description = *in.Description
		}
		if in.Data != nil {
			// ignore error as it's already validated in sanitize function
			t, _ := parseResolverType(*in.Data)
			original.Data = *in.Data
			original.Type = t
		}

		return nil
	})
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

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

	if in.Data != nil {
		_, err := parseResolverType(*in.Data)
		if err != nil {
			return err
		}
	}

	return nil
}
