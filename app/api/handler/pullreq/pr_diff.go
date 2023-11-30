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
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleDiff returns a http.HandlerFunc that returns diff.
func HandleDiff(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pullreqNumber, err := request.GetPullReqNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		setSHAs := func(sourceSHA, mergeBaseSHA string) {
			w.Header().Set("X-Source-Sha", sourceSHA)
			w.Header().Set("X-Merge-Base-Sha", mergeBaseSHA)
		}

		if strings.HasPrefix(r.Header.Get("Accept"), "text/plain") {
			err := pullreqCtrl.RawDiff(ctx, session, repoRef, pullreqNumber, setSHAs, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusOK)
			}
			return
		}

		_, includePatch := request.QueryParam(r, "include_patch")
		stream, err := pullreqCtrl.Diff(ctx, session, repoRef, pullreqNumber, setSHAs, includePatch)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSONArrayDynamic(ctx, w, stream)
	}
}
