// Copyright 2021 Harness Inc. All rights reserved.
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

/*
 * Writes json-encoded path information to the http response body.
 */
func HandleListPaths(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		spaceRef, err := request.GetSpaceRef(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pathFilter := request.ParsePathFilter(r)
		if pathFilter.Order == enum.OrderDefault {
			pathFilter.Order = enum.OrderAsc
		}

		paths, err := spaceCtrl.ListPaths(ctx, session, spaceRef, pathFilter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: do we need pagination? we should block that many paths in the first place.
		render.JSON(w, http.StatusOK, paths)
	}
}
