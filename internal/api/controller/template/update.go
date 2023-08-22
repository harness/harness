// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package template

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a template.
type UpdateInput struct {
	Description string `json:"description"`
	UID         string `json:"uid"`
	Data        string `json:"data"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	uid string,
	in *UpdateInput,
) (*types.Template, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckTemplate(ctx, c.authorizer, session, space.Path, uid, enum.PermissionTemplateEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	template, err := c.templateStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	return c.templateStore.UpdateOptLock(ctx, template, func(original *types.Template) error {
		if in.Description != "" {
			original.Description = in.Description
		}
		if in.Data != "" {
			original.Data = in.Data
		}
		if in.UID != "" {
			original.UID = in.UID
		}

		return nil
	})
}
