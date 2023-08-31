// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/execution"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

func HandleList(executionCtrl *execution.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineUID, err := request.GetPipelineUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pagination := request.ParsePaginationFromRequest(r)

		repos, totalCount, err := executionCtrl.List(ctx, session, repoRef, pipelineUID, pagination)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, pagination.Page, pagination.Size, int(totalCount))
		render.JSON(w, http.StatusOK, repos)
	}
}
