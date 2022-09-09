// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"encoding/json"
	"errors"
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
	"github.com/harness/gitness/types/errs"
	"github.com/rs/zerolog/hlog"
)

type spaceMoveRequest struct {
	Name        *string `json:"name"`
	ParentId    *int64  `json:"parentId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

/*
 * Moves an existing space.
 */
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
			if in.ParentId == nil {
				in.ParentId = &space.ParentId
			}

			// convert name to lower case for easy of api use
			*in.Name = strings.ToLower(*in.Name)

			// ensure we don't end up in any missconfiguration, and block no-ops
			if err = check.Name(*in.Name); err != nil {
				render.BadRequest(w, err)
				return
			} else if *in.ParentId == space.ParentId && *in.Name == space.Name {
				render.BadRequest(w, errs.NoChangeInRequestedMove)
				return
			}

			// TODO: restrict top level move
			// Ensure we can create spaces within the target space (using parent space as scope, similar to create)
			if *in.ParentId > 0 && *in.ParentId != space.ParentId {
				newParent, err := spaces.Find(ctx, *in.ParentId)
				if errors.Is(err, errs.ResourceNotFound) {
					render.NotFoundf(w, "Parent space not found.")
					return
				} else if err != nil {
					log.Err(err).
						Msgf("Failed to get target space with id %d for the move.", *in.ParentId)

					render.InternalError(w, errs.Internal)
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
				 * Validate path (Due to racing conditions we can't be 100% sure on the path here, but that's okay)
				 * Only needed if we actually change the parent (and can skip top level, as we already validate the name)
				 */
				path := paths.Concatinate(newParent.Path, *in.Name)
				if err = check.PathParams(path, true); err != nil {
					render.BadRequest(w, err)
					return
				}
			}

			res, err := spaces.Move(ctx, usr.ID, space.ID, *in.ParentId, *in.Name, in.KeepAsAlias)
			if errors.Is(err, errs.Duplicate) {
				log.Warn().Err(err).
					Msg("Failed to move the space as a duplicate was detected.")

				render.BadRequestf(w, "Unable to move the space as the destination path is already taken.")
				return
			} else if err != nil {
				log.Error().Err(err).
					Msg("Failed to move the space.")

				render.InternalError(w, errs.Internal)
				return
			}

			render.JSON(w, res, http.StatusOK)
		})
}
