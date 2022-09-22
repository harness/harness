// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type spaceMoveRequest struct {
	Name        *string `json:"name"`
	ParentID    *int64  `json:"parentId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

// HandleMove moves an existing space.
func HandleMove(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			usr, _ := request.UserFrom(ctx)
			space, _ := request.SpaceFrom(ctx)

			in := new(spaceMoveRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			// backfill data
			if in.Name == nil {
				in.Name = &space.Name
			}
			if in.ParentID == nil {
				in.ParentID = &space.ParentID
			}

			// convert name to lower case for easy of api use
			*in.Name = strings.ToLower(*in.Name)

			// ensure we don't end up in any missconfiguration, and block no-ops
			if err = check.Name(*in.Name); err != nil {
				render.UserfiedErrorOrInternal(w, err)
				return
			}

			if *in.ParentID == space.ParentID && *in.Name == space.Name {
				render.BadRequestError(w, render.ErrNoChange)
				return
			}

			// TODO: restrict top level move
			// Ensure we can create spaces within the target space (using parent space as scope, similar to create)
			if *in.ParentID > 0 && *in.ParentID != space.ParentID {
				var newParent *types.Space
				newParent, err = spaces.Find(ctx, *in.ParentID)
				if err != nil {
					log.Err(err).
						Msgf("Failed to get target space with id %d for the move.", *in.ParentID)

					render.UserfiedErrorOrInternal(w, err)
					return
				}

				scope := &types.Scope{SpacePath: newParent.Path}
				resource := &types.Resource{
					Type: enum.ResourceTypeSpace,
					Name: "",
				}
				if !guard.Enforce(w, r, scope, resource, enum.PermissionSpaceCreate) {
					return
				}

				/*
				 * Validate path length (Due to racing conditions we can't be 100% sure on the path here only best
				 * effort to avoid big transaction failure)
				 * Only needed if we actually change the parent (and can skip top level, as we already validate the name)
				 */
				path := paths.Concatinate(newParent.Path, *in.Name)
				if err = check.Path(path, true); err != nil {
					render.UserfiedErrorOrInternal(w, err)
					return
				}
			}

			res, err := spaces.Move(ctx, usr.ID, space.ID, *in.ParentID, *in.Name, in.KeepAsAlias)
			if err != nil {
				log.Error().Err(err).Msg("Failed to move the space.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, res)
		})
}
