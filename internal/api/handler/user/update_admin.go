// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleUpdateAdmin returns an http.HandlerFunc that processes an http.Request
// to update the current user admin status.
func HandleUpdateAdmin(userCtrl *user.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		userID, err := request.GetUserIDFromPath(r)
		if err != nil {
			render.BadRequestf(w, "Invalid request: %s.", err)
			return
		}

		in := new(user.UpdateAdminInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		user, err := userCtrl.UpdateAdmin(ctx, session, userID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, user)
	}
}
