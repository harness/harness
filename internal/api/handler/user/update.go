// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
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
		user, _ := request.UserFrom(ctx)

		in := new(types.UserInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		if in.Password != nil {
			hash, err := hashPassword([]byte(ptr.ToString(in.Password)), bcrypt.DefaultCost)
			if err != nil {
				log.Err(err).Msg("Failed to hash password.")

				render.InternalError(w)
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

		err = users.Update(ctx, user)
		if err != nil {
			log.Err(err).Msg("Failed to update the user.")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, user)
	}
}
