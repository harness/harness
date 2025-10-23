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

package gitspace

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/gitspace"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/paths"
)

func HandleEvents(gitspaceCtrl *gitspace.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		gitspaceRefFromPath, err := request.GetGitspaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		spaceRef, gitspaceIdentifier, err := paths.DisectLeaf(gitspaceRefFromPath)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		page := request.ParsePage(r)
		limit := request.ParseLimit(r)

		events, count, err := gitspaceCtrl.Events(ctx, session, spaceRef, gitspaceIdentifier, page, limit)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, page, limit, count)
		render.JSON(w, http.StatusOK, events)
	}
}
