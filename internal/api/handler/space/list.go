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

// HandleListSpaces writes json-encoded list of child spaces in the request body.
func HandleListSpaces(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		spaceFilter := request.ParseSpaceFilter(r)
		if spaceFilter.Order == enum.OrderDefault {
			spaceFilter.Order = enum.OrderAsc
		}

		spaces, totalCount, err := spaceCtrl.ListSpaces(ctx, session, spaceRef, spaceFilter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, spaceFilter.Page, spaceFilter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, spaces)
	}
}
