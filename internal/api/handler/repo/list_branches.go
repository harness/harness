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
 * Writes json-encoded branch information to the http response body.
 */
func HandleListBranches(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		includeCommit, err := request.GetIncludeCommitFromQueryOrDefault(r, false)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		branchFilter := request.ParseBranchFilter(r)

		branches, totalCount, err := repoCtrl.ListBranches(ctx, session, repoRef, includeCommit, branchFilter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, branchFilter.Page, branchFilter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, branches)
	}
}
