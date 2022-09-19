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

		expires := time.Now().Add(system.Config(ctx).Token.Expire)
		token, err := token.GenerateExp(user, expires.Unix(), user.Salt)
		if err != nil {
			log.Err(err).
				Str("user", username).
				Msg("failed to generate token")

			render.InternalError(w)
			return
		}

		// return the token if the with_user boolean
		// query parameter is set to true.
		if r.FormValue("return_user") == "true" {
			render.JSON(w, http.StatusOK,
				&types.UserToken{
					User: user,
					Token: &types.Token{
						Value:   token,
						Expires: expires.UTC(),
					},
				})
		} else {
			// else return the token only.
			render.JSON(w, http.StatusOK, &types.Token{Value: token})
		}
	}
}
