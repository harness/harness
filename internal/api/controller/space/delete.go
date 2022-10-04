// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

/*
* Delete deletes a space.
 */
func (c *Controller) Delete(ctx context.Context, session *auth.Session, spaceRef string) error {
	space, err := findSpaceFromRef(ctx, c.spaceStore, spaceRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceDelete, false); err != nil {
		return err
	}

	err = c.spaceStore.Delete(ctx, space.ID)
	if err != nil {
		return err
	}

	return nil
}
