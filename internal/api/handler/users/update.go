// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gotidy/ptr"
	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/types"
	"github.com/harness/scm/types/check"
	"github.com/rs/zerolog/hlog"

	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
)

// GenerateFromPassword returns the bcrypt hash of the
// password at the given cost.
var hashPassword = bcrypt.GenerateFromPassword

// HandleUpdate returns an http.HandlerFunc that processes an http.Request
// to update a user account.
func HandleUpdate(users store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		key := chi.URLParam(r, "user")
		user, err := users.FindKey(ctx, key)
		if err != nil {
			render.NotFound(w, err)
			log.Debug().Err(err).
				Str("user_key", key).
				Msg("cannot find user")
			return
		}

		in := new(types.UserInput)
		if err := json.NewDecoder(r.Body).Decode(in); err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Int64("user_id", user.ID).
				Str("user_email", user.Email).
				Msg("cannot unmarshal request")
			return
		}

		if in.Password != nil {
			hash, err := hashPassword([]byte(ptr.ToString(in.Password)), bcrypt.DefaultCost)
			if err != nil {
				render.InternalError(w, err)
				log.Debug().Err(err).
					Int64("user_id", user.ID).
					Str("user_email", user.Email).
					Msg("cannot hash password")
				return
			}
			user.Password = string(hash)
		}

		if in.Name != nil {
			user.Name = ptr.ToString(in.Name)
		}

		if in.Company != nil {
			user.Company = ptr.ToString(in.Company)
		}

		if in.Admin != nil {
			user.Admin = ptr.ToBool(in.Admin)
		}

		if in.Password != nil {
			hash, err := bcrypt.GenerateFromPassword([]byte(ptr.ToString(in.Password)), bcrypt.DefaultCost)
			if err != nil {
				render.InternalError(w, err)
				log.Debug().Err(err).
					Msg("cannot hash password")
				return
			}
			user.Password = string(hash)
		}

		if ok, err := check.User(user); !ok {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Int64("user_id", user.ID).
				Str("user_email", user.Email).
				Msg("cannot update user")
			return
		}

		user.Updated = time.Now().UnixMilli()

		err = users.Update(ctx, user)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Int64("user_id", user.ID).
				Str("user_email", user.Email).
				Msg("cannot update user")
		} else {
			render.JSON(w, user, 200)
		}
	}
}
