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
)

func HandleUpdatePipeline(aiagentCtrl *aiagent.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// TODO Question: how come we decode body here vs putting that logic in the request package?
		in := new(aiagent.UpdatePipelineInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		pipeline, err := aiagentCtrl.UpdatePipeline(ctx, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pipeline))
	}
}
