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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

// HandleList writes json-encoded list of child spaces in the request body.
func HandleList(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceView,
		true,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			space, _ := request.SpaceFrom(ctx)

			params := request.ParseSpaceFilter(r)
			if params.Order == enum.OrderDefault {
				params.Order = enum.OrderAsc
			}

			count, err := spaces.Count(ctx, space.ID)
			if err != nil {
				log.Err(err).Msgf("Failed to count child spaces.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			allSpaces, err := spaces.List(ctx, space.ID, params)
			if err != nil {
				log.Err(err).Msgf("Failed to list child spaces.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			/*
			 * Only list spaces that are either public or can be accessed by the user
			 *
			 * TODO: optimize by making a single auth check for all spaces at once.
			 * TODO: maybe ommit permission check for performance.
			 * TODO: count is off in case not all repos are accessible.
			 */
			result := make([]*types.Space, 0, len(allSpaces))
			for _, cs := range allSpaces {
				if !cs.IsPublic {
					err = guard.CheckSpace(r, enum.PermissionSpaceView, cs.Path)
					if err != nil {
						log.Debug().Err(err).
							Msgf("Skip space '%s' in output.", cs.Path)
						continue
					}
				}

				result = append(result, cs)
			}

			render.Pagination(r, w, params.Page, params.Size, int(count))
			render.JSON(w, http.StatusOK, result)
		})
}
