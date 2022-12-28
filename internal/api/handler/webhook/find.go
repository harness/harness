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

// HandleFind returns a http.HandlerFunc that finds a webhook.
func HandleFind(webhookCtrl *webhook.Controller) http.HandlerFunc {
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

		webhook, err := webhookCtrl.Find(ctx, session, repoRef, webhookID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, webhook)
	}
}
