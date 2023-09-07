// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/logs"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

func HandleFind(logCtrl *logs.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		pipelineUID, err := request.GetPipelineUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		executionNum, err := request.GetExecutionNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		stageNum, err := request.GetStageNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		stepNum, err := request.GetStepNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		lines, err := logCtrl.Find(
			ctx, session, repoRef, pipelineUID,
			executionNum, int(stageNum), int(stepNum))
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, lines)
	}
}
