// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

// HandleListSessionTokens returns an http.HandlerFunc that
// writes a json-encoded list of Tokens to the http.Response body.
func HandleListSessionTokens(tokenStore store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := hlog.FromRequest(r)
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)

		res, err := tokenStore.List(ctx, user.ID, enum.TokenTypeSession)
		if err != nil {
			log.Err(err).Msg("failed to list sessions")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, res)
	}
}
