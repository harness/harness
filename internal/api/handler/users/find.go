// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/store"
	"github.com/rs/zerolog/hlog"

	"github.com/go-chi/chi"
)

// HandleFind returns an http.HandlerFunc that writes json-encoded
// user account information to the the response body.
func HandleFind(users store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		key := chi.URLParam(r, "user")
		user, err := users.FindKey(ctx, key)
		if err != nil {
			log.Debug().Err(err).Msgf("Failed to get user using key '%s'.", key)

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, user)
	}
}
