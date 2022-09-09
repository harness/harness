// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/comms"
	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/errs"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type repoMoveRequest struct {
	Name        *string `json:"name"`
	SpaceId     *int64  `json:"spaceId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

/*
 * Moves an existing repo.
 */
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
			if in.SpaceId == nil {
				in.SpaceId = &repo.SpaceId
			}

			// convert name to lower case for easy of api use
			*in.Name = strings.ToLower(*in.Name)

			// ensure we don't end up in any missconfiguration, and block no-ops
			if err = check.Name(*in.Name); err != nil {
				render.BadRequest(w, err)
				return
			} else if *in.SpaceId == repo.SpaceId && *in.Name == repo.Name {
				render.BadRequest(w, errs.NoChangeInRequestedMove)
				return
			} else if *in.SpaceId <= 0 {
				render.BadRequest(w, check.ErrRepositoryRequiresSpaceId)
				return
			}

			// Ensure we have access to the target space (if its a space move)
			if *in.SpaceId != repo.SpaceId {
				newSpace, err := spaces.Find(ctx, *in.SpaceId)
				if errors.Is(err, errs.ResourceNotFound) {
					render.NotFoundf(w, "Parent space not found.")
					return
				} else if err != nil {
					log.Err(err).
						Msgf("Failed to get target space with id %d for the move.", *in.SpaceId)

					render.InternalErrorf(w, comms.Internal)
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

			res, err := repos.Move(ctx, usr.ID, repo.ID, *in.SpaceId, *in.Name, in.KeepAsAlias)
			if errors.Is(err, errs.Duplicate) {
				log.Warn().Err(err).
					Msg("Failed to move the repo as a duplicate was detected.")

				render.BadRequestf(w, "Unable to move the repository as the destination path is already taken.")
				return
			} else if err != nil {
				log.Error().Err(err).
					Msg("Failed to move the repository.")

				render.InternalErrorf(w, comms.Internal)
				return
			}

			render.JSON(w, res, http.StatusOK)
		})
}
