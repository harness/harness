// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type createPatRequest struct {
	Name     string           `json:"name"`
	LifeTime time.Duration    `json:"lifetime"`
	Grants   enum.AccessGrant `json:"grants"`
}

// HandleCreatePAT returns an http.HandlerFunc that creates a new PAT and
// writes a json-encoded TokenResponse to the http.Response body.
func HandleCreatePAT(tokenStore store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := hlog.FromRequest(r)
		ctx := r.Context()
		principal, _ := request.PrincipalFrom(ctx)
		user, _ := request.UserFrom(ctx)

		in := new(createPatRequest)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		token, jwtToken, err := token.CreatePAT(ctx, tokenStore, principal, user, in.Name, in.LifeTime, in.Grants)
		if err != nil {
			log.Err(err).Msg("failed to create pat")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, &types.TokenResponse{Token: *token, AccessToken: jwtToken})
	}
}
