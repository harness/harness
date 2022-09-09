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
 * Deletes a space.
 */
func HandleDelete(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceDelete,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			s, _ := request.SpaceFrom(ctx)

			err := spaces.Delete(r.Context(), s.ID)
			if errors.Is(err, errs.ResourceNotFound) {
				render.NotFoundf(w, "Space not found.")
				return
			} else if err != nil {
				log.Err(err).Msgf("Failed to delete the space.")

				render.InternalErrorf(w, comms.Internal)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
}
