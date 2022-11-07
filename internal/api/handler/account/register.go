// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/render"
)

// HandleRegister returns an http.HandlerFunc that processes an http.Request
// to register the named user account with the system.
func HandleRegister(userCtrl *user.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		in := &user.CreateInput{
			UID:         r.FormValue("username"),
			DisplayName: r.FormValue("displayname"),
			Email:       r.FormValue("email"),
			Password:    r.FormValue("password"),
		}

		tokenResponse, err := userCtrl.Register(ctx, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, tokenResponse)
	}
}
