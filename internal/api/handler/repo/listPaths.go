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
	"github.com/rs/zerolog/hlog"
)

/*
 * Writes json-encoded path information to the http response body.
 */
func HandleListPaths(guard *guard.Guard, repos store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoView,
		true,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			repo, _ := request.RepoFrom(ctx)

			params := request.ParsePathFilter(r)
			if params.Order == enum.OrderDefault {
				params.Order = enum.OrderAsc
			}

			paths, err := repos.ListAllPaths(ctx, repo.ID, params)
			if err != nil {
				log.Err(err).Msgf("Failed to get list of repo paths.")

				render.InternalError(w)
				return
			}

			// TODO: do we need pagination? we should block that many paths in the first place.
			render.JSON(w, http.StatusOK, paths)
		})
}
