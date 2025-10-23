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

package repo

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

/*
 * Writes json-encoded commit information to the http response body.
 */
func HandleListCommits(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		gitRef := request.GetGitRefFromQueryOrDefault(r, "")

		filter, err := request.ParseCommitFilter(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		list, err := repoCtrl.ListCommits(ctx, session, repoRef, gitRef, filter)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		// TODO: get last page indicator explicitly - current check is wrong in case len % limit == 0
		isLastPage := len(list.Commits) < filter.Limit
		render.PaginationNoTotal(r, w, filter.Page, filter.Limit, isLastPage)
		render.JSON(w, http.StatusOK, list)
	}
}
