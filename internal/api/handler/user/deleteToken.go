// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types/enum"
)

// HandleDeleteToken returns an http.HandlerFunc that
// deletes a token of a user.
func HandleDeleteToken(userCtrl *user.Controller, tokenType enum.TokenType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		userUID := session.Principal.UID

		tokenID, err := request.GetTokenIDFromPath(r)
		if err != nil {
			render.BadRequest(w)
			return
		}

		err = userCtrl.DeleteToken(ctx, session, userUID, tokenType, tokenID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
