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

// HandleList returns a http.HandlerFunc that lists webhooks.
func HandleList(webhookCtrl *webhook.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter := request.ParseWebhookFilter(r)

		webhooks, totalCount, err := webhookCtrl.List(ctx, session, repoRef, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, webhooks)
	}
}
