// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

// HandleDeleteSession returns an http.HandlerFunc that
// deletes a session token of a user.
func HandleDeleteSession(tokenStore store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := hlog.FromRequest(r)
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)

		sessionTokenID, err := request.GetSessionTokenID(r)
		if err != nil {
			render.BadRequest(w)
			return
		}

		// Ensure pat belongs to us!
		token, err := tokenStore.Find(ctx, sessionTokenID)
		if errors.Is(err, store.ErrResourceNotFound) {
			render.UserfiedErrorOrInternal(w, err)
			return
		}

		if token.Type != enum.TokenTypeSession || token.PrincipalID != user.ID {
			log.Warn().Msg("User tried to delete token that doesn't belong to themselves or is no session token")

			// render not found - no need for user to know other token ids.
			render.NotFound(w)
			return
		}

		err = tokenStore.Delete(ctx, sessionTokenID)
		if err != nil {
			log.Err(err).Msg("failed to delete session")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
