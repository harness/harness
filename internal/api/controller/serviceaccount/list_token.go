// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListTokens lists all tokens of a service account.
func (c *Controller) ListTokens(ctx context.Context, session *auth.Session,
	saUID string) ([]*types.Token, error) {
	sa, err := findServiceAccountFromUID(ctx, c.principalStore, saUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent (ensures that parent exists)
	if err = apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		sa.ParentType, sa.ParentID, sa.UID, enum.PermissionServiceAccountView); err != nil {
		return nil, err
	}

	return c.tokenStore.List(ctx, sa.ID, enum.TokenTypeSAT)
}
