// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types"
)

// HandleCommits returns commits for PR.
func HandleCommits(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
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

		filter := &types.PaginationFilter{
			Page:  request.ParsePage(r),
			Limit: request.ParseLimit(r),
		}

		// gitref is Head branch in this case
		commits, err := pullreqCtrl.Commits(ctx, session, repoRef, pullreqNumber, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: get last page indicator explicitly - current check is wrong in case len % limit == 0
		isLastPage := len(commits) < filter.Limit
		render.PaginationNoTotal(r, w, filter.Page, filter.Limit, isLastPage)
		render.JSON(w, http.StatusOK, commits)
	}
}
