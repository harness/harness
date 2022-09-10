// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"

	"github.com/dchest/uniuri"
)

type userCreateInput struct {
	Username string `json:"email"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

// HandleCreate returns an http.HandlerFunc that processes an http.Request
// to create the named user account in the system.
func HandleCreate(users store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		in := new(userCreateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Err(err).
				Str("email", in.Username).
				Msg("Failed to hash password")

			render.InternalError(w)
			return
		}

		user := &types.User{
			Email:    in.Username,
			Admin:    in.Admin,
			Password: string(hash),
			Salt:     uniuri.NewLen(uniuri.UUIDLen),
			Created:  time.Now().UnixMilli(),
			Updated:  time.Now().UnixMilli(),
		}

		if ok, err := check.User(user); !ok {
			log.Debug().Err(err).
				Str("email", user.Email).
				Msg("invalid user input")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		err = users.Create(ctx, user)
		if err != nil {
			log.Err(err).
				Str("email", user.Email).
				Msg("failed to create user")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, user)
	}
}
