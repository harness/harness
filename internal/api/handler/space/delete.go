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
	"github.com/rs/zerolog/log"
)

/*
 * Deletes a space.
 */
func HandleDelete(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceDelete,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			// TODO: return 200 if space confirmed doesn't exist

			ctx := r.Context()
			s, _ := request.SpaceFrom(ctx)

			err := spaces.Delete(r.Context(), s.ID)
			if err != nil {
				render.InternalError(w, err)
				log.Error().Err(err).
					Str("space_fqn", s.Fqn).
					Msg("Failed to delete space.")
				return

			}

			w.WriteHeader(http.StatusNoContent)
		})
}
