// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
 * ListTokens lists all tokens of a user.
 */
func (c *Controller) ListTokens(ctx context.Context, session *auth.Session,
	userUID string, tokenType enum.TokenType) ([]*types.Token, error) {
	user, err := findUserFromUID(ctx, c.userStore, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserView); err != nil {
		return nil, err
	}

	if !isUserTokenType(tokenType) {
		return nil, usererror.ErrBadRequest
	}

	return c.tokenStore.List(ctx, user.ID, tokenType)
}
