// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types/enum"
)

// HandleList returns a http.HandlerFunc that lists pull requests for a repository.
func HandleList(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter, err := request.ParsePullReqFilter(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		if filter.Order == enum.OrderDefault {
			filter.Order = enum.OrderDesc
		}

		list, total, err := pullreqCtrl.List(ctx, session, repoRef, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(total))
		render.JSON(w, http.StatusOK, list)
	}
}
