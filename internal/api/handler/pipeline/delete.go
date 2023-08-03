// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

/*
 * Deletes a pipeline.
 */
func HandleDelete(pipelineCtrl *pipeline.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineRef, err := request.GetPipelinePathRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, pipelineUID, err := SplitRef(pipelineRef)
		if err != nil {
			render.TranslatedUserError(w, err)
		}

		err = pipelineCtrl.Delete(ctx, session, spaceRef, pipelineUID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
