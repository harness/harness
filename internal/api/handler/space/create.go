// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type spaceCreateInput struct {
	Name        string `json:"name"`
	ParentId    int64  `json:"parentId"`
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

		in := new(spaceCreateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("Decoding json body failed.")
			return
		}

		// Get fqn and parentFqn
		parentFqn := ""
		fqn := in.Name
		if in.ParentId > 0 {
			parentSpace, err := spaces.Find(ctx, in.ParentId)
			if err != nil {
				render.BadRequest(w, err)
				log.Debug().
					Err(err).
					Msgf("Parent space '%s' doesn't exist.", parentFqn)

				return
			}

			// parentFqn is assumed to be valid, in.Name gets validated in check.Space function
			parentFqn = parentSpace.Fqn
			fqn = parentFqn + "/" + in.Name
		}

		// get current user (will be enforced to not be nil via explicit check or guard.Enforce)
		usr, _ := request.UserFrom(ctx)

		/*
		 * AUTHORIZATION
		 * Can only be done once we know the parent space
		 */
		if in.ParentId <= 0 {
			// TODO: Restrict top level space creation.
			if usr == nil {
				render.Unauthorized(w, errors.New("Authentication required."))
				return
			}
		} else {
			// Create is a special case - check permission without specific resource
			scope := &types.Scope{SpaceFqn: parentFqn}
			resource := &types.Resource{
				Type: enum.ResourceTypeSpace,
				Name: "",
			}
			if !guard.Enforce(w, r, scope, resource, enum.PermissionSpaceCreate) {
				return
			}
		}

		space := &types.Space{
			Name:        strings.ToLower(in.Name),
			ParentId:    in.ParentId,
			Fqn:         strings.ToLower(fqn),
			DisplayName: in.DisplayName,
			Description: in.Description,
			IsPublic:    in.IsPublic,
			CreatedBy:   usr.ID,
			Created:     time.Now().UnixMilli(),
			Updated:     time.Now().UnixMilli(),
		}

		if ok, err := check.Space(space); !ok {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("Space validation failed.")
			return
		}

		err = spaces.Create(ctx, space)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Msg("Space creation failed")
		} else {
			render.JSON(w, space, 200)
		}
	}
}
