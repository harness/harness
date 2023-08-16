// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package secret

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Delete(ctx context.Context, session *auth.Session, spaceRef string, uid string) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return fmt.Errorf("could not find space: %w", err)
	}

	err = apiauth.CheckSecret(ctx, c.authorizer, session, space.Path, uid, enum.PermissionSecretDelete)
	if err != nil {
		return fmt.Errorf("failed to authorize: %w", err)
	}
	err = c.secretStore.DeleteByUID(ctx, space.ID, uid)
	if err != nil {
		return fmt.Errorf("could not delete secret: %w", err)
	}
	return nil
}
