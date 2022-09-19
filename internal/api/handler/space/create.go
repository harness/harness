// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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

type spaceCreateRequest struct {
	Name        string `json:"name"`
	ParentID    int64  `json:"parentId"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
}

/*
 * HandleCreate returns an http.HandlerFunc that creates a new space.
 */
func HandleCreate(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		in := new(spaceCreateRequest)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		// get current user (will be enforced to not be nil via explicit check or guard.Enforce)
		usr, _ := request.UserFrom(ctx)

		// Collect parent path along the way - needed for duplicate error message
		parentPath := ""

		/*
		 * AUTHORIZATION
		 * Can only be done once we know the parent space
		 */
		if in.ParentID <= 0 {
			// TODO: Restrict top level space creation.
			if usr == nil {
				render.Unauthorized(w)
				return
			}
		} else {
			// Create is a special case - we need the parent path
			var parent *types.Space
			parent, err = spaces.Find(ctx, in.ParentID)
			if err != nil {
				log.Err(err).Msgf("Failed to get space with id '%d'.", in.ParentID)

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			scope := &types.Scope{SpacePath: parent.Path}
			resource := &types.Resource{
				Type: enum.ResourceTypeSpace,
				Name: "",
			}
			if !guard.Enforce(w, r, scope, resource, enum.PermissionSpaceCreate) {
				return
			}

			parentPath = parent.Path
		}

		// create new space object
		space := &types.Space{
			Name:        strings.ToLower(in.Name),
			ParentID:    in.ParentID,
			DisplayName: in.DisplayName,
			Description: in.Description,
			IsPublic:    in.IsPublic,
			CreatedBy:   usr.ID,
			Created:     time.Now().UnixMilli(),
			Updated:     time.Now().UnixMilli(),
		}

		// validate space
		if err = check.Space(space); err != nil {
			render.UserfiedErrorOrInternal(w, err)
			return
		}

		// Validate path length (Due to racing conditions we can't be 100% sure on the path here only best effort
		// to have a quick failure)
		path := paths.Concatinate(parentPath, space.Name)
		if err = check.Path(path, true); err != nil {
			render.UserfiedErrorOrInternal(w, err)
			return
		}

		// create in store
		err = spaces.Create(ctx, space)
		if err != nil {
			log.Error().Err(err).
				Msg("Space creation failed.")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, space)
	}
}
