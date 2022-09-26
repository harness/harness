// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin returns an http.HandlerFunc that authenticates
// the user and returns an authentication token on success.
func HandleLogin(userStore store.UserStore, system store.SystemStore, tokenStore store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		username := r.FormValue("username")
		password := r.FormValue("password")
		user, err := userStore.FindEmail(ctx, username)
		if err != nil {
			log.Debug().Err(err).
				Str("user", username).
				Msg("cannot find user")

			// always give not found error as extra security measurement.
			render.NotFound(w)
			return
		}

		err = bcrypt.CompareHashAndPassword(
			[]byte(user.Password),
			[]byte(password),
		)
		if err != nil {
			log.Debug().Err(err).
				Str("user", username).
				Msg("invalid password")

			render.NotFound(w)
			return
		}

		token, jwtToken, err := token.CreateUserSession(ctx, tokenStore, user, "login")
		if err != nil {
			log.Err(err).
				Str("user", username).
				Msg("failed to generate token")

			render.InternalError(w)
			return
		}

		render.JSON(w, http.StatusOK, &types.TokenResponse{Token: *token, AccessToken: jwtToken})
	}
}
