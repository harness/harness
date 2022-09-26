// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dchest/uniuri"
	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/handler/common"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * Creates a new service account and writes json-encoded service account to the http response body.
 */
func HandleCreate(guard *guard.Guard, saStore store.ServiceAccountStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		in := new(common.CreateServiceAccountRequest)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		sa := &types.ServiceAccount{
			Name:       in.Name,
			Salt:       uniuri.NewLen(uniuri.UUIDLen),
			Created:    time.Now().UnixMilli(),
			Updated:    time.Now().UnixMilli(),
			ParentType: in.ParentType,
			ParentID:   in.ParentID,
		}

		// validate service account
		if err = check.ServiceAccount(sa); err != nil {
			render.UserfiedErrorOrInternal(w, err)
			return
		}

		// Ensure principal has required permissions on parent (ensures that parent exists)
		if !guard.EnforceInParentScope(w, r,
			&types.Resource{Type: enum.ResourceTypeServiceAccount, Name: ""},
			enum.PermissionServiceAccountCreate, sa.ParentType, sa.ParentID) {
			return
		}

		// TODO: Racing condition with parent (space/repo) being deleted!
		err = saStore.Create(ctx, sa)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create service account.")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, sa)
	}
}
