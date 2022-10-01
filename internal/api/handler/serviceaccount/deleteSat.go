// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/guard"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

// HandleDeleteSAT returns an http.HandlerFunc that
// deletes a SAT token of a service account.
func HandleDeleteSAT(guard *guard.Guard, tokenStore store.TokenStore) http.HandlerFunc {
	return guard.ServiceAccount(
		enum.PermissionServiceAccountEdit,
		func(w http.ResponseWriter, r *http.Request) {
			log := hlog.FromRequest(r)
			ctx := r.Context()
			sa, _ := request.ServiceAccountFrom(ctx)

			satID, err := request.GetSatID(r)
			if err != nil {
				render.BadRequest(w)
				return
			}

			token, err := tokenStore.Find(ctx, satID)
			if err != nil {
				render.UserfiedErrorOrInternal(w, err)
				return
			}

			// Ensure sat belongs to service account
			if token.Type != enum.TokenTypeSAT || token.PrincipalID != sa.ID {
				log.Warn().Msg("Principal tried to delete token that doesn't belong to the service account")

				// render not found - no need for principal to know other token ids.
				render.NotFound(w)
				return
			}

			err = tokenStore.Delete(ctx, satID)
			if err != nil {
				log.Err(err).Msg("failed to delete SAT")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
