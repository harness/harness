// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/errs"
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
		if errors.Is(err, errs.ResourceNotFound) {
			render.NotFoundf(w, "User not found.")
			return
		} else if err != nil {
			log.Err(err).Msgf("Failed to get user using key '%s'.", key)

			render.InternalError(w, errs.Internal)
			return
		}

		err = users.Delete(ctx, user)
		if err != nil {
			log.Error().Err(err).
				Int64("user_id", user.ID).
				Str("user_email", user.Email).
				Msg("failed to delete user")

			render.InternalError(w, err)
			return

		}

		w.WriteHeader(http.StatusNoContent)
	}
}
