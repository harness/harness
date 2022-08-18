// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"net/http"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/api/request"
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/internal/token"
	"github.com/harness/scm/types"
	"github.com/rs/zerolog/hlog"
)

// HandleToken returns an http.HandlerFunc that generates and
// writes a json-encoded token to the http.Response body.
func HandleToken(users store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, _ := request.UserFrom(r.Context())

		token, err := token.Generate(viewer, viewer.Salt)
		if err != nil {
			render.InternalErrorf(w, "Failed to generate token")
			hlog.FromRequest(r).
				Error().Err(err).
				Str("user", viewer.Email).
				Msg("failed to generate token")
			return
		}

		render.JSON(w, &types.Token{Value: token}, 200)
	}
}
