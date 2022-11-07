// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleDeleteToken returns an http.HandlerFunc that
// deletes a SAT token of a service account.
func HandleDeleteToken(saCrl *serviceaccount.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		saUID, err := request.GetServiceAccountUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		tokenUID, err := request.GetTokenUIDFromPath(r)
		if err != nil {
			render.BadRequest(w)
			return
		}

		err = saCrl.DeleteToken(ctx, session, saUID, tokenUID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
