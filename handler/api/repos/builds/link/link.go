// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package link

import (
	"net/http"
	"strconv"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"

	"github.com/go-chi/chi"
)

// payload wraps the link and returns to the
// client as a valid json object.
type payload struct {
	Link string `json:"link"`
}

// HandleLink returns an http.HandlerFunc that redirects the
// user to the git resource in the remote source control
// management system.
func HandleLink(
	repos core.RepositoryStore,
	builds core.BuildStore,
	linker core.Linker,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx       = r.Context()
			namespace = chi.URLParam(r, "owner")
			name      = chi.URLParam(r, "name")
		)
		number, err := strconv.ParseInt(chi.URLParam(r, "number"), 10, 64)
		if err != nil {
			render.BadRequest(w, err)
			return
		}
		repo, err := repos.FindName(ctx, namespace, name)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		build, err := builds.FindNumber(ctx, repo.ID, number)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		to, err := linker.Link(ctx, repo, build)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		if r.FormValue("redirect") == "true" {
			http.Redirect(w, r, to, http.StatusSeeOther)
			return
		}
		v := &payload{to}
		render.JSON(w, v, http.StatusOK)
	}
}
