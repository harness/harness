// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/harness/gitness/types/errs"
	"github.com/rs/zerolog/hlog"
)

/*
 * Writes json-encoded list of repos in the request body.
 */
func HandleListRepos(guard *guard.Guard, repos store.RepoStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceView,
		true,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			space, _ := request.SpaceFrom(ctx)

			params := request.ParseRepoFilter(r)
			if params.Order == enum.OrderDefault {
				params.Order = enum.OrderAsc
			}

			count, err := repos.Count(ctx, space.ID)
			if err != nil {
				log.Err(err).Msgf("Failed to count child repos.")

				render.InternalError(w, errs.Internal)
				return
			}

			allRepos, err := repos.List(ctx, space.ID, params)
			if err != nil {
				log.Err(err).Msgf("Failed to list child repos.")

				render.InternalError(w, errs.Internal)
				return
			}

			/*
			 * Only list repos that are either public or can be accessed by the user
			 *
			 * TODO: optimize by making a single auth check for all repos at once.
			 */
			result := make([]*types.Repository, 0, len(allRepos))
			for _, rep := range allRepos {
				if !rep.IsPublic {
					err := guard.CheckRepo(r, enum.PermissionRepoView, rep.Path)
					if err != nil {
						log.Debug().Err(err).
							Msgf("Skip repo '%s' in output.", rep.Path)

						continue
					}
				}

				result = append(result, rep)
			}

			render.Pagination(r, w, params.Page, params.Size, int(count))
			render.JSON(w, result, 200)
		})
}
