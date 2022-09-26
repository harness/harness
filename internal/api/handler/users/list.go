// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

// HandleList returns an http.HandlerFunc that writes a json-encoded
// list of all registered system users to the response body.
func HandleList(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		params := request.ParseUserFilter(r)
		if params.Order == enum.OrderDefault {
			params.Order = enum.OrderAsc
		}

		count, err := userStore.Count(ctx)
		if err != nil {
			log.Err(err).
				Msg("Failed to retrieve user count")
		}

		list, err := userStore.List(ctx, params)
		if err != nil {
			log.Err(err).
				Msg("Failed to retrieve user list")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.Pagination(r, w, params.Page, params.Size, int(count))
		render.JSON(w, http.StatusOK, list)
	}
}
