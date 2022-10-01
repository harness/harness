// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/guard"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * Writes json-encoded service account information to the http response body.
 */
func HandleListServiceAccounts(guard *guard.Guard, saStore store.ServiceAccountStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionServiceAccountView,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			repo, _ := request.RepoFrom(ctx)

			sas, err := saStore.List(ctx, enum.ParentResourceTypeRepo, repo.ID)
			if err != nil {
				log.Err(err).Msgf("Failed to get list of service accounts for repo.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			// TODO: do we need pagination? we should block that many service accounts in the first place.
			render.JSON(w, http.StatusOK, sas)
		})
}
