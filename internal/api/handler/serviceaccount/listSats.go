// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

// HandleListSATs returns an http.HandlerFunc that
// writes a json-encoded list of Tokens to the http.Response body.
func HandleListSATs(guard *guard.Guard, tokenStore store.TokenStore) http.HandlerFunc {
	return guard.ServiceAccount(
		enum.PermissionServiceAccountView,
		func(w http.ResponseWriter, r *http.Request) {
			log := hlog.FromRequest(r)
			ctx := r.Context()
			sa, _ := request.ServiceAccountFrom(ctx)

			res, err := tokenStore.List(ctx, sa.ID, enum.TokenTypeSAT)
			if err != nil {
				log.Err(err).Msg("failed to list SATs")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, res)
		})
}
