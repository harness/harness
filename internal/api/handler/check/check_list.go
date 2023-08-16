// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/check"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleCheckList is an HTTP handler for listing status check results for a repository.
func HandleCheckList(checkCtrl *check.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		commitSHA, err := request.GetCommitSHAFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		opts := request.ParseCheckListOptions(r)

		checks, count, err := checkCtrl.ListChecks(ctx, session, repoRef, commitSHA, opts)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, opts.Page, opts.Size, count)
		render.JSON(w, http.StatusOK, checks)
	}
}
