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

package user

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
)

// HandleDeleteFavorite returns a http.HandlerFunc that delete a favorite.
func HandleDeleteFavorite(userCtrl *user.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		resourceID, err := request.GetResourceIDFromPath(r)
		if err != nil {
			render.BadRequest(ctx, w)
			return
		}

		resourceType, err := request.ParseResourceType(r)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid query param: %s.", err)
			return
		}

		err = userCtrl.DeleteFavorite(ctx, session, &types.FavoriteResource{Type: resourceType, ID: resourceID})
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
