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
 * Deletes a given path.
 */
func HandleDeletePath(guard *guard.Guard, repoStore store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			repo, _ := request.RepoFrom(ctx)

			pathID, err := request.GetPathID(r)
			if err != nil {
				render.BadRequest(w)
				return
			}

			err = repoStore.DeletePath(ctx, repo.ID, pathID)
			if err != nil {
				log.Err(err).Int64("path_id", pathID).
					Msgf("Failed to delete repo path.")

				render.UserfiedErrorOrInternal(w, err)
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
