// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin returns an http.HandlerFunc that authenticates
// the user and returns an authentication token on success.
func HandleLogin(users store.UserStore, system store.SystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		username := r.FormValue("username")
		password := r.FormValue("password")
		user, err := users.FindEmail(ctx, username)
		if err != nil {
			log.Debug().Err(err).
				Str("user", username).
				Msg("cannot find user")

			// always give not found error as extra security measurement.
			render.NotFoundf(w, "Invalid email or password")
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

			render.NotFoundf(w, "Invalid email or password")
			return
		}

		expires := time.Now().Add(system.Config(ctx).Token.Expire)
		token_, err := token.GenerateExp(user, expires.Unix(), user.Salt)
		if err != nil {
			log.Err(err).
				Str("user", username).
				Msg("failed to generate token")

			render.InternalErrorf(w, "Failed to create session")
			return
		}

		// return the token if the with_user boolean
		// query parameter is set to true.
		if r.FormValue("return_user") == "true" {
			render.JSON(w, &types.UserToken{
				User: user,
				Token: &types.Token{
					Value:   token_,
					Expires: expires.UTC(),
				},
			}, 200)
		} else {
			// else return the token only.
			render.JSON(w, &types.Token{Value: token_}, 200)
		}
	}
}
