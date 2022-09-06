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
	"github.com/rs/zerolog/log"
)

/*
 * Deletes a repository.
 */
func HandleDelete(guard *guard.Guard, repos store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoDelete,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			// TODO: return 200 if repo confirmed doesn't exist

			ctx := r.Context()
			rep, _ := request.RepoFrom(ctx)

			err := repos.Delete(r.Context(), rep.ID)
			if err != nil {
				render.InternalError(w, err)
				log.Error().Err(err).
					Int64("repo_id", rep.ID).
					Str("repo_fqn", rep.Fqn).
					Msg("Failed to delete repository.")
				return

			}

			w.WriteHeader(http.StatusNoContent)
		})
}
