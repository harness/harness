// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/principal"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleFind returns an http.HandlerFunc that writes json-encoded
// principal information to the http response body.
func HandleFind(principalCtrl *principal.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pUID, err := request.GetPrincipalUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pInfo, err := principalCtrl.FindInfoByUIDNoAuth(ctx, pUID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, pInfo)
	}
}
