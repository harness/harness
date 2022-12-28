// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleListExecutions returns a http.HandlerFunc that lists webhook executions.
func HandleListExecutions(webhookCtrl *webhook.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		webhookID, err := request.GetWebhookIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter := request.ParseWebhookExecutionFilter(r)

		executions, err := webhookCtrl.ListExecutions(ctx, session, repoRef, webhookID, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// TODO: get last page indicator explicitly - current check is wrong in case len % pageSize == 0
		isLastPage := len(executions) < filter.Size
		render.PaginationNoTotal(r, w, filter.Page, filter.Size, isLastPage)
		render.JSON(w, http.StatusOK, executions)
	}
}
