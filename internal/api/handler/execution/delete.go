// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/execution"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/paths"
)

func HandleDelete(executionCtrl *execution.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineRef, err := request.GetPipelineRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, pipelineUID, err := paths.DisectLeaf(pipelineRef)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		n, err := request.GetExecutionNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		err = executionCtrl.Delete(ctx, session, spaceRef, pipelineUID, n)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
