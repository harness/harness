// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types"
)

func HandleListSecrets(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		filter := request.ParseSecretFilter(r)
		ret, totalCount, err := spaceCtrl.ListSecrets(ctx, session, spaceRef, filter)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// Strip out data in the returned value
		secrets := []types.Secret{}
		for _, s := range ret {
			secrets = append(secrets, *s.CopyWithoutData())
		}

		render.Pagination(r, w, filter.Page, filter.Size, int(totalCount))
		render.JSON(w, http.StatusOK, secrets)
	}
}
