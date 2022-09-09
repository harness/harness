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

	"github.com/harness/gitness/internal/api/comms"
	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/handler/common"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/errs"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * Writes json-encoded path information to the http response body.
 */
func HandleCreatePath(guard *guard.Guard, spaces store.SpaceStore) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)
			space, _ := request.SpaceFrom(ctx)
			usr, _ := request.UserFrom(ctx)

			in := new(common.CreatePathRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			params := &types.PathParams{
				Path:      strings.ToLower(in.Path),
				CreatedBy: usr.ID,
				Created:   time.Now().UnixMilli(),
				Updated:   time.Now().UnixMilli(),
			}

			// validate path
			if err = check.PathParams(params.Path, true); err != nil {
				render.BadRequest(w, err)
				return
			}

			// TODO: ensure user is authorized to create a path pointing to in.Path
			path, err := spaces.CreatePath(ctx, space.ID, params)
			if errors.Is(err, errs.Duplicate) {
				log.Warn().Err(err).
					Msg("Failed to create path for space as a duplicate was detected.")

				render.BadRequestf(w, "Path '%s' already exists.", params.Path)
				return
			} else if err != nil {
				log.Error().Err(err).
					Msg("Failed to create path for space.")

				render.InternalErrorf(w, comms.Internal)
				return
			}

			render.JSON(w, path, 200)
		})
}
