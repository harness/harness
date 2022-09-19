// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type repoMoveRequest struct {
	Name        *string `json:"name"`
	SpaceID     *int64  `json:"spaceId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

// HandleMove moves an existing repo.
func HandleMove(guard *guard.Guard, repos store.RepoStore, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			usr, _ := request.UserFrom(ctx)
			repo, _ := request.RepoFrom(ctx)

			in := new(repoMoveRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			// backfill data
			if in.Name == nil {
				in.Name = &repo.Name
			}
			if in.SpaceID == nil {
				in.SpaceID = &repo.SpaceID
			}

			// convert name to lower case for easy of api use
			*in.Name = strings.ToLower(*in.Name)

			// ensure we don't end up in any missconfiguration, and block no-ops
			if err = check.Name(*in.Name); err != nil {
				render.UserfiedErrorOrInternal(w, err)
				return
			}
			if *in.SpaceID == repo.SpaceID && *in.Name == repo.Name {
				render.BadRequestError(w, render.ErrNoChange)
				return
			}
			if *in.SpaceID <= 0 {
				render.UserfiedErrorOrInternal(w, check.ErrRepositoryRequiresSpaceID)
				return
			}

			// Ensure we have access to the target space (if its a space move)
			if *in.SpaceID != repo.SpaceID {
				var newSpace *types.Space
				newSpace, err = spaces.Find(ctx, *in.SpaceID)
				if err != nil {
					log.Err(err).Msgf("Failed to get target space with id %d for the move.", *in.SpaceID)

					render.UserfiedErrorOrInternal(w, err)
					return
				}

				// Ensure we can create repos within the space (using space as scope, similar to create)
				scope := &types.Scope{SpacePath: newSpace.Path}
				resource := &types.Resource{
					Type: enum.ResourceTypeRepo,
					Name: "",
				}
				if !guard.Enforce(w, r, scope, resource, enum.PermissionRepoCreate) {
					return
				}
			}

			res, err := repos.Move(ctx, usr.ID, repo.ID, *in.SpaceID, *in.Name, in.KeepAsAlias)
			if err != nil {
				log.Error().Err(err).Msg("Failed to move the repository.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, res)
		})
}
