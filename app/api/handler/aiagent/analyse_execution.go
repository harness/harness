// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aiagent

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/app/api/controller/aiagent"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
)

func HandleAnalyse(aiagentCtrl *aiagent.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := new(types.AnalyseExecutionInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		pipelineIdentifier := in.PipelineIdentifier
		if pipelineIdentifier == "" {
			render.TranslatedUserError(ctx, w, usererror.BadRequest("pipeline_identifier is required"))
			return
		}
		repoRef := in.RepoRef
		if repoRef == "" {
			render.TranslatedUserError(ctx, w, usererror.BadRequest("repo_ref is required"))
			return
		}

		executionNumber := in.ExecutionNum
		if executionNumber < 1 {
			render.TranslatedUserError(ctx, w, usererror.BadRequest("execution_number must be greater than 0"))
			return
		}

		// fetch stored analysis for the given execution, if any
		analysis, err := aiagentCtrl.GetAnalysis(ctx, session, repoRef, pipelineIdentifier, in.ExecutionNum)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		render.JSON(w, http.StatusCreated, analysis)
	}
}
