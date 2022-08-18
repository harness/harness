// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"net/http"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/store"
	"github.com/rs/zerolog/hlog"

	"github.com/go-chi/chi"
)

// HandleDelete returns an http.HandlerFunc that processes an http.Request
// to delete the named user account from the system.
func HandleDelete(users store.UserStore) http.HandlerFunc {
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
		err = users.Delete(ctx, user)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Int64("user_id", user.ID).
				Str("user_email", user.Email).
				Msg("cannot delete user")

		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
