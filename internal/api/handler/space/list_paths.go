// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types/enum"
)

// HandleListPaths writes json-encoded path information to the http response body.
func HandleListPaths(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter := request.ParsePathFilter(r)
		if filter.Order == enum.OrderDefault {
			filter.Order = enum.OrderAsc
		}

		paths, err := spaceCtrl.ListPaths(ctx, session, spaceRef, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: do we need pagination? we should block that many paths in the first place.
		render.JSON(w, http.StatusOK, paths)
	}
}
