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

	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleFind returns a http.HandlerFunc that finds a pull request.
func HandleFind(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
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

		options, err := request.ParsePullReqMetadataOptions(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		pr, err := pullreqCtrl.Find(ctx, session, repoRef, pullreqNumber, options)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, pr)
	}
}

// HandleFindByBranches returns a http.HandlerFunc that finds a pull request from the provided branch pair.
func HandleFindByBranches(pullreqCtrl *pullreq.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		sourceRepoRef := request.GetSourceRepoRefFromQueryOrDefault(r, repoRef)

		sourceBranch, err := request.GetPullReqSourceBranchFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		targetBranch, err := request.GetPullReqTargetBranchFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		options, err := request.ParsePullReqMetadataOptions(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		pr, err := pullreqCtrl.FindByBranches(ctx, session, repoRef, sourceRepoRef, sourceBranch, targetBranch, options)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, pr)
	}
}
