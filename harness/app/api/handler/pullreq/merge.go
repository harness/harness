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

package pullreq

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleMerge returns a http.HandlerFunc that merges the pull request.
func HandleMerge(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(pullreq.MergeInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil && !errors.Is(err, io.EOF) { // allow empty body
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		pullreqNumber, err := request.GetPullReqNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		pr, violation, err := pullreqCtrl.Merge(ctx, session, repoRef, pullreqNumber, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		if violation != nil {
			render.Unprocessable(w, violation)
			return
		}

		render.JSON(w, http.StatusOK, pr)
	}
}
