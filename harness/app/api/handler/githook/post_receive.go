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
)

// HandlePostReceive returns a handler function that handles post-receive git hooks.
func HandlePostReceive(
	githookCtrl *controllergithook.Controller,
	git git.Interface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := types.GithookPostReceiveInput{}
		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}
		// Harness doesn't require any custom git connector.
		out, err := githookCtrl.PostReceive(ctx, git, session, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, out)
	}
}
