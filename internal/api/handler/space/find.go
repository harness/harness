// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/space"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

/*
 * Writes json-encoded space information to the http response body.
 */
func HandleFind(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceView,
		func(w http.ResponseWriter, r *http.Request) {
			s, sref, err := space.UsingRefParam(r, spaces)
			if err != nil {
				render.BadRequest(w, err)
				log.Debug().Err(err).
					Str("space_sref", sref).
					Msg("Failed to get space.")
			}

			render.JSON(w, s, 200)
		})
}
