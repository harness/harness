// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/trigger"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

func HandleList(triggerCtrl *trigger.Controller) http.HandlerFunc {
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

		filter := request.ParseListQueryFilterFromRequest(r)

		repos, totalCount, err := triggerCtrl.List(ctx, session, repoRef, pipelineUID, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, repos)
	}
}
