// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

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
 * Deletes a given path.
 */
func HandleDeletePath(guard *guard.Guard, spaceStore store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			space, _ := request.SpaceFrom(ctx)

			pathID, err := request.GetPathID(r)
			if err != nil {
				render.BadRequest(w)
				return
			}

			err = spaceStore.DeletePath(ctx, space.ID, pathID)
			if err != nil {
				log.Err(err).Int64("path_id", pathID).
					Msgf("Failed to delete space path.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
