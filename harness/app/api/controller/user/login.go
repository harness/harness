// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user

import (
	"context"
	"errors"

	"github.com/harness/gitness/app/api/usererror"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/token"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	LoginIdentifier string `json:"login_identifier"`
	Password        string `json:"password"`
}

// Login attempts to login as a specific user - returns the session token if successful.
func (c *Controller) Login(
	ctx context.Context,
	in *LoginInput,
) (*types.TokenResponse, error) {
	// no auth check required, password is used for it.

	user, err := findUserFromUID(ctx, c.principalStore, in.LoginIdentifier)
	if errors.Is(err, store.ErrResourceNotFound) {
		user, err = findUserFromEmail(ctx, c.principalStore, in.LoginIdentifier)
	}

	// always return not found for security reasons.
	if err != nil {
		log.Ctx(ctx).Debug().Err(err).
			Msgf("failed to retrieve user %q during login (returning ErrNotFound).", in.LoginIdentifier)
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

	tokenIdentifier := token.GenerateIdentifier("login")

	token, jwtToken, err := token.CreateUserSession(ctx, c.tokenStore, user, tokenIdentifier)
	if err != nil {
		return nil, err
	}

	c.eventReporter.LoggedIn(ctx, &userevents.LoggedInPayload{
		Base: userevents.Base{PrincipalID: user.ID},
	})

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}
