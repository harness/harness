// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"encoding/json"
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/api/request"
	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/types"
	"github.com/rs/zerolog/hlog"

	"github.com/gotidy/ptr"
	"golang.org/x/crypto/bcrypt"
)

// GenerateFromPassword returns the bcrypt hash of the
// password at the given cost.
var hashPassword = bcrypt.GenerateFromPassword

// HandleUpdate returns an http.HandlerFunc that processes an http.Request
// to update the current user account.
func HandleUpdate(users store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)
		viewer, _ := request.UserFrom(ctx)

		in := new(types.UserInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Error().Err(err).
				Str("email", viewer.Email).
				Msg("cannot unmarshal request")
			return
		}

		if in.Password != nil {
			hash, err := hashPassword([]byte(ptr.ToString(in.Password)), bcrypt.DefaultCost)
			if err != nil {
				render.InternalError(w, err)
				log.Debug().Err(err).
					Msg("cannot hash password")
				return
			}
			viewer.Password = string(hash)
		}

		if in.Name != nil {
			viewer.Name = ptr.ToString(in.Name)
		}

		if in.Company != nil {
			viewer.Company = ptr.ToString(in.Company)
		}

		err = users.Update(ctx, viewer)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Str("email", viewer.Email).
				Msg("cannot update user")
		} else {
			render.JSON(w, viewer, 200)
		}
	}
}
