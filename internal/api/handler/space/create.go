// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/space"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type spaceCreateInput struct {
	Description string `json:"description"`
}

/*
 * HandleCreate returns an http.HandlerFunc that creates a new space.
 */
func HandleCreate(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		// get fqsn (requires parent if child space)
		sref, err := space.GetRefParam(r)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Failed to get the fqsn.")
			return
		}

		// Assume sref is fqsn and disect (if it's ID validation will fail later)
		parentFqsn, name, err := types.DisectFqn(sref)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Failed to desict sref '%s'.", sref)
			return
		}

		/*
		 * AUTHORIZATION
		 * Can only be done once we know the parent space
		 * TODO: Restrict top level space creation.
		 */
		if parentFqsn != "" && !guard.CheckSpace(w, r, enum.PermissionRepoCreate, parentFqsn) {
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
		parentId := int64(-1)
		if parentFqsn != "" {
			parentSpace, err := spaces.FindFqsn(ctx, parentFqsn)
			if err != nil {
				render.BadRequest(w, err)
				log.Debug().
					Err(err).
					Msgf("Parent space '%s' doesn't exist.", parentFqsn)
				return
			}

			parentId = parentSpace.ID
		}

		space := &types.Space{
			Name:        name,
			ParentId:    parentId,
			Fqsn:        sref,
			Description: in.Description,
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
