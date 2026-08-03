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

package keywordsearch

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/keywordsearch"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleSearchSpace returns keyword search results for repositories in a single space.
func HandleSearchSpace(ctrl *keywordsearch.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		query := request.ParseQuery(r)
		limit := request.ParseLimit(r)

		recursive, err := request.ParseRecursiveFromQuery(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		regex, err := request.ParseRegexFromQuery(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		repoPaths := request.GetRepoPathsFromQuery(r)

		result, err := ctrl.SearchSpace(ctx, session, spaceRef, keywordsearch.SearchSpaceInput{
			Query:     query,
			RepoPaths: repoPaths,
			Limit:     limit,
			Regex:     regex,
			Recursive: recursive,
		})
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, result)
	}
}
