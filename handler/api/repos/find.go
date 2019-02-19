// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package repos

import (
	"net/http"

	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
)

// HandleFind returns an http.HandlerFunc that writes the
// json-encoded repository details to the response body.
func HandleFind() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		repo, _ := request.RepoFrom(ctx)
		perm, _ := request.PermFrom(ctx)
		repo.Perms = perm
		render.JSON(w, repo, 200)
	}
}
