// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

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
 * Deletes a repository.
 */
func HandleDelete(guard *guard.Guard, repoStore store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoDelete,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			repo, _ := request.RepoFrom(ctx)

			err := repoStore.Delete(r.Context(), repo.ID)
			if err != nil {
				log.Err(err).Msgf("Failed to delete the Repository.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
