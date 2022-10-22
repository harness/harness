// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types/enum"
)

/*
 * Writes json-encoded path information to the http response body.
 */
func HandleListPaths(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter := request.ParsePathFilter(r)
		if filter.Order == enum.OrderDefault {
			filter.Order = enum.OrderAsc
		}

		paths, err := repoCtrl.ListPaths(ctx, session, repoRef, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: implement pagination - or should we block that many paths in the first place.
		render.JSON(w, http.StatusOK, paths)
	}
}
