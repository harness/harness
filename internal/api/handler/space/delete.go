// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

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
 * Deletes a space.
 */
func HandleDelete(guard *guard.Guard, spaceStore store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceDelete,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			s, _ := request.SpaceFrom(ctx)

			err := spaceStore.Delete(r.Context(), s.ID)
			if err != nil {
				log.Err(err).Msgf("Failed to delete the space.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
