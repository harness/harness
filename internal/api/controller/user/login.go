// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"errors"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

/*
 * Login attempts to login as a specific user - returns the session token if successful.
 */
func (c *Controller) Login(ctx context.Context, session *auth.Session,
	username string, password string) (*types.TokenResponse, error) {
	// no auth check required, password is used for it.

	user, err := findUserFromUID(ctx, c.userStore, username)
	if errors.Is(err, store.ErrResourceNotFound) {
		user, err = findUserFromEmail(ctx, c.userStore, username)
	}

	// always return not found for security reasons.
	if err != nil {
		log.Ctx(ctx).Debug().Err(err).
			Str("user_uid", username).
			Msgf("failed to retrieve user during login.")
		return nil, usererror.ErrNotFound
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	)
	if err != nil {
		log.Debug().Err(err).
			Str("user_uid", user.UID).
			Msg("invalid password")

		return nil, usererror.ErrNotFound
	}

	// TODO: how should we name session tokens?
	token, jwtToken, err := token.CreateUserSession(ctx, c.tokenStore, user, "login")
	if err != nil {
		return nil, err
	}

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}
