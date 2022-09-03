// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/space"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

/*
 * Writes json-encoded list of child spaces in the request body.
 */
func HandleList(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
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

			ctx := r.Context()

			params := request.ParseSpaceFilter(r)
			if params.Order == enum.OrderDefault {
				params.Order = enum.OrderAsc
			}

			count, err := spaces.Count(ctx, s.ID)
			if err != nil {
				render.InternalError(w, err)
				log.Error().Err(err).
					Msgf("Failed to retrieve count of spaces under '%s'.", s.Fqsn)
				return
			}

			list, err := spaces.List(ctx, s.ID, params)
			if err != nil {
				render.InternalError(w, err)
				log.Error().Err(err).
					Msgf("Failed to retrieve list of spaces under '%s'.", s.Fqsn)
				return
			}

			render.Pagination(r, w, params.Page, params.Size, int(count))
			render.JSON(w, list, 200)
		})
}
