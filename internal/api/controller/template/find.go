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

func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	uid string,
) (*types.Template, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("could not find space: %w", err)
	}
	err = apiauth.CheckTemplate(ctx, c.authorizer, session, space.Path, uid, enum.PermissionTemplateView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}
	template, err := c.templateStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("could not find template: %w", err)
	}
	return template, nil
}
