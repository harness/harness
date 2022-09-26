// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/rs/zerolog/hlog"
)

// HandleDelete returns an http.HandlerFunc that processes an http.Request
// to delete the named user account from the system.
func HandleDelete(userStore store.UserStore, tokenStore store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)
		user, _ := request.UserFrom(ctx)

		// delete all tokens (okay if we fail after - user is tried to being deleted anyway)
		err := tokenStore.DeleteForPrincipal(ctx, user.ID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete tokens for user.")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		err = userStore.Delete(ctx, user)
		if err != nil {
			log.Error().Err(err).
				Int64("user_id", user.ID).
				Str("user_email", user.Email).
				Msg("failed to delete user")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
