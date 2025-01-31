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
	"io"
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/errors"
	gittypes "github.com/harness/gitness/git/api"
)

// HandleDiff returns a http.HandlerFunc that returns diff.
func HandleDiff(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		pullreqNumber, err := request.GetPullReqNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		setSHAs := func(sourceSHA, mergeBaseSHA string) {
			w.Header().Set("X-Source-Sha", sourceSHA)
			w.Header().Set("X-Merge-Base-Sha", mergeBaseSHA)
		}
		files := gittypes.FileDiffRequests{}

		switch r.Method {
		case http.MethodPost:
			if err = json.NewDecoder(r.Body).Decode(&files); err != nil && !errors.Is(err, io.EOF) {
				render.TranslatedUserError(ctx, w, err)
				return
			}
		case http.MethodGet:
			// TBD: this will be removed in future because of URL limit in browser to 2048 chars.
			files = request.GetFileDiffFromQuery(r)
		}

		if strings.HasPrefix(r.Header.Get("Accept"), "text/plain") {
			err := pullreqCtrl.RawDiff(ctx, w, session, repoRef, pullreqNumber, setSHAs, files...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusOK)
			}
			return
		}

		includePatch, err := request.QueryParamAsBoolOrDefault(r, "include_patch", false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		ignoreWhitespace, err := request.QueryParamAsBoolOrDefault(r, request.QueryParamIgnoreWhitespace, false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		stream, err := pullreqCtrl.Diff(
			ctx,
			session,
			repoRef,
			pullreqNumber,
			setSHAs,
			includePatch,
			ignoreWhitespace,
			files...,
		)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSONArrayDynamic(ctx, w, stream)
	}
}
