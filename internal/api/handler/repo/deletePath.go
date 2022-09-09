// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/harness/gitness/types/errs"
	"github.com/rs/zerolog/hlog"
)

/*
 * Deletes a given path.
 */
func HandleDeletePath(guard *guard.Guard, repos store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			repo, _ := request.RepoFrom(ctx)

			pathId, err := request.GetPathId(r)
			if err != nil {
				render.BadRequest(w, err)
				return
			}

			err = repos.DeletePath(ctx, repo.ID, pathId)
			if errors.Is(err, errs.ResourceNotFound) {
				render.NotFoundf(w, "Path doesn't exist.")
				return
			} else if errors.Is(err, errs.PrimaryPathCantBeDeleted) {
				render.BadRequestf(w, "Deleting a primary path is not allowed.")
				return
			} else if err != nil {
				log.Err(err).Int64("path_id", pathId).
					Msgf("Failed to delete repo path.")

				render.InternalError(w, errs.Internal)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
