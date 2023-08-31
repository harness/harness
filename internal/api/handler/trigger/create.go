// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/trigger"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

func HandleCreate(triggerCtrl *trigger.Controller) http.HandlerFunc {
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

		in := new(trigger.CreateInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		trigger, err := triggerCtrl.Create(ctx, session, repoRef, pipelineUID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusCreated, trigger)
	}
}
