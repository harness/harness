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

package space

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types/enum"
)

// HandleListSpaces writes json-encoded list of child spaces in the request body.
func HandleListSpaces(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		spaceFilter, err := request.ParseSpaceFilter(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		if spaceFilter.Order == enum.OrderDefault {
			spaceFilter.Order = enum.OrderAsc
		}

		spaces, totalCount, err := spaceCtrl.ListSpaces(ctx, session, spaceRef, spaceFilter)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.Pagination(r, w, spaceFilter.Page, spaceFilter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, spaces)
	}
}
