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

package githook

import (
	"encoding/json"
	"net/http"

	controllergithook "github.com/harness/gitness/app/api/controller/githook"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// HandleUpdate returns a handler function that handles update git hooks.
func HandleUpdate(
	githookCtrl *controllergithook.Controller,
	git git.Interface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := types.GithookUpdateInput{}
		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("internal request body in update githook")
			render.BadRequest(ctx, w)
			return
		}

		out, err := githookCtrl.Update(ctx, git, session, in)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("internal server error in update githook")
			render.InternalError(ctx, w)
			return
		}

		render.JSON(w, http.StatusOK, out)
	}
}
