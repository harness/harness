// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleUpdate returns a http.HandlerFunc that updates an existing webhook.
func HandleUpdate(webhookCtrl *webhook.Controller) http.HandlerFunc {
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

		in := new(webhook.UpdateInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		hook, err := webhookCtrl.Update(ctx, session, repoRef, webhookID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, hook)
	}
}
