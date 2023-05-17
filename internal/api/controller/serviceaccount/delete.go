// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// Delete deletes a service account.
func (c *Controller) Delete(ctx context.Context, session *auth.Session,
	saUID string) error {
	sa, err := findServiceAccountFromUID(ctx, c.principalStore, saUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent (ensures that parent exists)
	if err = apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		sa.ParentType, sa.ParentID, sa.UID, enum.PermissionServiceAccountDelete); err != nil {
		return err
	}

	// delete all tokens (okay if we fail after - user intends to delete service account anyway)
	// TODO: cascading delete?
	err = c.tokenStore.DeleteForPrincipal(ctx, sa.ID)
	if err != nil {
		return fmt.Errorf("failed to delete tokens for service account: %w", err)
	}

	return c.principalStore.DeleteServiceAccount(ctx, sa.ID)
}
