// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog/hlog"
)

// HandleToken returns an http.HandlerFunc that generates and
// writes a json-encoded token to the http.Response body.
func HandleToken(users store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := hlog.FromRequest(r)
		user, _ := request.UserFrom(r.Context())

		token, err := token.Generate(user, user.Salt)
		if err != nil {
			log.Err(err).Msg("failed to generate token")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, &types.Token{Value: token})
	}
}
