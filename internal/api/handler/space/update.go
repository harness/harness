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
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

type spaceUpdateRequest struct {
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"isPublic"`
}

/*
 * Updates an existing space.
 */
func HandleUpdate(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			space, _ := request.SpaceFrom(ctx)

			in := new(spaceUpdateRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			// update values only if provided
			if in.DisplayName != nil {
				space.DisplayName = *in.DisplayName
			}
			if in.Description != nil {
				space.Description = *in.Description
			}
			if in.IsPublic != nil {
				space.IsPublic = *in.IsPublic
			}

			// always update time
			space.Updated = time.Now().UnixMilli()

			// ensure provided values are valid
			if err := check.Space(space); err != nil {
				render.UserfiedErrorOrInternal(w, err)
				return
			}

			err = spaces.Update(ctx, space)
			if err != nil {
				log.Error().Err(err).Msg("Space update failed.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, space)
		})
}
