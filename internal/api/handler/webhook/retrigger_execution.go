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

// HandleRetriggerExecution returns a http.HandlerFunc that retriggers a webhook executions.
func HandleRetriggerExecution(webhookCtrl *webhook.Controller) http.HandlerFunc {
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

		webhookExecutionID, err := request.GetWebhookExecutionIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		execution, err := webhookCtrl.RetriggerExecution(ctx, session, repoRef, webhookID, webhookExecutionID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, execution)
	}
}
