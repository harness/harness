// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleListActivities returns a http.HandlerFunc that lists pull request activities for a pull request.
func HandleListActivities(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pullreqNumber, err := request.GetPullReqNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter, err := request.ParsePullReqActivityFilter(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		list, total, err := pullreqCtrl.ListActivities(ctx, session, repoRef, pullreqNumber, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.PaginationLimit(r, w, int(total))
		render.JSON(w, http.StatusOK, list)
	}
}
