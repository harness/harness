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
	PathName    *string `json:"pathName"`
	SpaceID     *int64  `json:"spaceId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

// HandleMove moves an existing repo.
//nolint:gocognit,goimports // exception for now, one of the more complicated parts of the code
func HandleMove(guard *guard.Guard, repoStore store.RepoStore, spaceStore store.SpaceStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			principal, _ := request.PrincipalFrom(ctx)
			repo, _ := request.RepoFrom(ctx)

			in := new(repoMoveRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			// backfill data
			if in.PathName == nil {
				in.PathName = &repo.PathName
			}
			if in.SpaceID == nil {
				in.SpaceID = &repo.SpaceID
			}

			// convert name to lower case for easy of api use
			*in.PathName = strings.ToLower(*in.PathName)

			// ensure we don't end up in any missconfiguration, and block no-ops
			if err = check.PathName(*in.PathName); err != nil {
				render.UserfiedErrorOrInternal(w, err)
				return
			}
			if *in.SpaceID == repo.SpaceID && *in.PathName == repo.PathName {
				render.BadRequestError(w, render.ErrNoChange)
				return
			}
			if *in.SpaceID <= 0 {
				render.UserfiedErrorOrInternal(w, check.ErrRepositoryRequiresSpaceID)
				return
			}

			// Ensure we have access to the target space (if it's a space move)
			if *in.SpaceID != repo.SpaceID {
				var newSpace *types.Space
				newSpace, err = spaceStore.Find(ctx, *in.SpaceID)
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

			res, err := repoStore.Move(ctx, principal.ID, repo.ID, *in.SpaceID, *in.PathName, in.KeepAsAlias)
			if err != nil {
				log.Error().Err(err).Msg("Failed to move the repository.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, res)
		})
}
