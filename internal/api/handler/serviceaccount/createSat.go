// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type createSatRequest struct {
	Name     string           `json:"name"`
	LifeTime time.Duration    `json:"lifetime"`
	Grants   enum.AccessGrant `json:"grants"`
}

// HandleCreateSAT returns an http.HandlerFunc that creates a new SAT and
// writes a json-encoded TokenResponse to the http.Response body.
func HandleCreateSAT(guard *guard.Guard, tokenStore store.TokenStore) http.HandlerFunc {
	return guard.ServiceAccount(
		enum.PermissionServiceAccountEdit,
		func(w http.ResponseWriter, r *http.Request) {
			log := hlog.FromRequest(r)
			ctx := r.Context()
			principal, _ := request.PrincipalFrom(ctx)

			in := new(createSatRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			// We need the service account for which the SAT gets created, differs from executing principal
			sa, _ := request.ServiceAccountFrom(ctx)

			token, jwtToken, err := token.CreateSAT(ctx, tokenStore, principal,
				sa, in.Name, in.LifeTime, in.Grants)
			if err != nil {
				log.Err(err).Msg("failed to create sat")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, &types.TokenResponse{Token: *token, AccessToken: jwtToken})
		})
}
