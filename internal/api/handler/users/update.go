// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleUpdate returns a http.HandlerFunc that processes an http.Request
// to update a user account.
func HandleUpdate(userCtrl *user.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		userUID, err := request.GetUserUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		in := new(user.UpdateInput)
		if err = json.NewDecoder(r.Body).Decode(in); err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		usr, err := userCtrl.Update(ctx, session, userUID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, usr)
	}
}
