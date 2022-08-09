// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"net/http"
	"time"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/internal/token"
	"github.com/bradrydzewski/my-app/types"
	"github.com/bradrydzewski/my-app/types/check"

	"github.com/dchest/uniuri"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"
)

// HandleRegister returns an http.HandlerFunc that processes an http.Request
// to register the named user account with the system.
func HandleRegister(users store.UserStore, system store.SystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		username := r.FormValue("username")
		password := r.FormValue("password")

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			render.InternalError(w, err)
			log.Debug().Err(err).
				Str("email", username).
				Msg("cannot hash password")
			return
		}

		user := &types.User{
			Name:     username,
			Email:    username,
			Password: string(hash),
			Salt:     uniuri.NewLen(uniuri.UUIDLen),
			Created:  time.Now().UnixMilli(),
			Updated:  time.Now().UnixMilli(),
		}

		if ok, err := check.User(user); !ok {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Str("email", username).
				Msg("invalid user input")
			return
		}

		if err := users.Create(ctx, user); err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Str("email", username).
				Msg("cannot create user")
			return
		}

		// if the registered user is the first user of the system,
		// assume they are the system administrator and grant the
		// user system admin access.
		if user.ID == 1 {
			user.Admin = true
			if err := users.Update(ctx, user); err != nil {
				log.Error().Err(err).
					Str("email", username).
					Msg("cannot enable admin user")
			}
		}

		expires := time.Now().Add(system.Config(ctx).Token.Expire)
		token_, err := token.GenerateExp(user, expires.Unix(), user.Salt)
		if err != nil {
			render.InternalErrorf(w, "Failed to create session")
			log.Error().Err(err).
				Str("email", username).
				Msg("failed to generate token")
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
