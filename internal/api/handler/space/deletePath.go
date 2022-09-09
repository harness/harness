// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/comms"
	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/errs"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * Deletes a given path.
 */
func HandleDeletePath(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			space, _ := request.SpaceFrom(ctx)

			pathId, err := request.GetPathId(r)
			if err != nil {
				render.BadRequest(w, err)
				return
			}

			err = spaces.DeletePath(ctx, space.ID, pathId)
			if errors.Is(err, errs.ResourceNotFound) {
				render.NotFoundf(w, "Path doesn't exist.")
				return
			} else if errors.Is(err, errs.PrimaryPathCantBeDeleted) {
				render.BadRequestf(w, "Deleting a primary path is not allowed.")
				return
			} else if err != nil {
				log.Err(err).Int64("path_id", pathId).
					Msgf("Failed to delete space path.")

				render.InternalErrorf(w, comms.Internal)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
