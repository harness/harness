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

		// get fqn (requires parent of child space)
		sref, err := request.GetSpaceRef(r)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Failed to get the fqn.")
			return
		}

		// Assume sref is fqn and disect (if it's ID validation will fail later)
		parentFqn, name, err := types.DisectFqn(sref)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Failed to desict sref '%s'.", sref)
			return
		}

		// get current user
		usr, _ := request.UserFrom(ctx)

		/*
		 * AUTHORIZATION
		 * Can only be done once we know the parent space
		 *
		 * TODO: Restrict top level space creation.
		 */
		if parentFqn == "" && usr == nil {
			render.Unauthorized(w, errors.New("Authentication required."))
			return
		} else if !guard.EnforceSpace(w, r, enum.PermissionRepoCreate, parentFqn) {
			return
		}

		in := new(spaceCreateInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("Decoding json body failed.")
			return
		}

		// get parentId if needed
		parentId := int64(0)
		if parentFqn != "" {
			parentSpace, err := spaces.FindFqn(ctx, parentFqn)
			if err != nil {
				render.BadRequest(w, err)
				log.Debug().
					Err(err).
					Msgf("Parent space '%s' doesn't exist.", parentFqn)
				return
			}

			parentId = parentSpace.ID
		}

		space := &types.Space{
			Name:        strings.ToLower(name),
			ParentId:    parentId,
			Fqn:         strings.ToLower(sref),
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
