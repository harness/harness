// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
)

/*
 * Writes json-encoded repository information to the http response body.
 */
func HandleFind(guard *guard.Guard, repos store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoView,
		true,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			repo, _ := request.RepoFrom(ctx)

			render.JSON(w, http.StatusOK, repo)
		})
}
