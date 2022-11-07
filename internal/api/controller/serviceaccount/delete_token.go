// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

/*
 * DeleteToken deletes a token of a sevice account.
 */
func (c *Controller) DeleteToken(ctx context.Context, session *auth.Session,
	saUID string, tokenUID string) error {
	sa, err := findServiceAccountFromUID(ctx, c.saStore, saUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent (ensures that parent exists)
	if err = apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		sa.ParentType, sa.ParentID, sa.UID, enum.PermissionServiceAccountEdit); err != nil {
		return err
	}

	token, err := c.tokenStore.FindByUID(ctx, sa.ID, tokenUID)
	if err != nil {
		return err
	}

	// Ensure sat belongs to service account
	if token.Type != enum.TokenTypeSAT || token.PrincipalID != sa.ID {
		log.Warn().Msg("Principal tried to delete token that doesn't belong to the service account")

		// throw a not found error - no need for user to know about token?
		return usererror.ErrNotFound
	}

	return c.tokenStore.Delete(ctx, token.ID)
}
