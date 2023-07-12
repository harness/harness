// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

/*
 * Login attempts to login as a specific user - returns the session token if successful.
 */
func (c *Controller) Login(ctx context.Context, session *auth.Session,
	in *LoginInput) (*types.TokenResponse, error) {
	// no auth check required, password is used for it.

	user, err := findUserFromUID(ctx, c.principalStore, in.Username)
	if errors.Is(err, store.ErrResourceNotFound) {
		user, err = findUserFromEmail(ctx, c.principalStore, in.Username)
	}

	// always return not found for security reasons.
	if err != nil {
		log.Ctx(ctx).Debug().Err(err).
			Str("user_uid", in.Username).
			Msgf("failed to retrieve user during login.")
		return nil, usererror.ErrNotFound
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(in.Password),
	)
	if err != nil {
		log.Debug().Err(err).
			Str("user_uid", user.UID).
			Msg("invalid password")

		return nil, usererror.ErrNotFound
	}

	tokenUID, err := generateSessionTokenUID()
	if err != nil {
		return nil, err
	}
	token, jwtToken, err := token.CreateUserSession(ctx, c.tokenStore, user, tokenUID)
	if err != nil {
		return nil, err
	}

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}

func generateSessionTokenUID() (string, error) {
	r, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}
	return fmt.Sprintf("login-%d-%04d", time.Now().Unix(), r.Int64()), nil
}
