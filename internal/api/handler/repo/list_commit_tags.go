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
 * Writes json-encoded commit tag information to the http response body.
 */
func HandleListCommitTags(repoCtrl *repo.Controller) http.HandlerFunc {
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

		filter := request.ParseTagFilter(r)

		tags, err := repoCtrl.ListCommitTags(ctx, session, repoRef, includeCommit, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: get last page indicator explicitly - current check is wrong in case len % pageSize == 0
		isLastPage := len(tags) < filter.Size
		render.PaginationNoTotal(r, w, filter.Page, filter.Size, isLastPage)
		render.JSON(w, http.StatusOK, tags)
	}
}
