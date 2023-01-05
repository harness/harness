// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

/*
 * DeleteToken deletes a token of a user.
 */
func (c *Controller) DeleteToken(ctx context.Context, session *auth.Session,
	userUID string, tokenType enum.TokenType, tokenUID string) error {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return err
	}

	token, err := c.tokenStore.FindByUID(ctx, user.ID, tokenUID)
	if err != nil {
		return err
	}

	// Ensure token type matches the requested type and is a valid user token type
	if !isUserTokenType(token.Type) || token.Type != tokenType {
		// throw a not found error - no need for user to know about token.
		return usererror.ErrNotFound
	}

	// Ensure token belongs to user.
	if token.PrincipalID != user.ID {
		log.Warn().Msg("Principal tried to delete token that doesn't belong to the user")

		// throw a not found error - no need for user to know about token.
		return usererror.ErrNotFound
	}

	return c.tokenStore.Delete(ctx, token.ID)
}
