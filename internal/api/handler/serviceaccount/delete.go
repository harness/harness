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

/*
 * Deletes a service account.
 */
func HandleDelete(guard *guard.Guard, saStore store.ServiceAccountStore, tokenStore store.TokenStore) http.HandlerFunc {
	return guard.ServiceAccount(
		enum.PermissionServiceAccountDelete,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			sa, _ := request.ServiceAccountFrom(ctx)

			// delete all tokens (okay if we fail after - user intends to delete service account anyway)
			err := tokenStore.DeleteForPrincipal(ctx, sa.ID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to delete tokens for service account.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			err = saStore.Delete(ctx, sa.ID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to delete the service account.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
