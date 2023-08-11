// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

/*
 * Writes json-encoded commit information to the http response body.
 */
func HandleListCommits(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		gitRef := request.GetGitRefFromQueryOrDefault(r, "")

		filter, err := request.ParseCommitFilter(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		list, err := repoCtrl.ListCommits(ctx, session, repoRef, gitRef, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: get last page indicator explicitly - current check is wrong in case len % limit == 0
		isLastPage := len(list.Commits) < filter.Limit
		render.PaginationNoTotal(r, w, filter.Page, filter.Limit, isLastPage)
		render.JSON(w, http.StatusOK, list)
	}
}
