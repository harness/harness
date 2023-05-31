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

// HandleCreateAccessToken returns an http.HandlerFunc that creates a new PAT and
// writes a json-encoded TokenResponse to the http.Response body.
func HandleCreateAccessToken(userCtrl *user.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		userUID := session.Principal.UID

		in := new(user.CreateTokenInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid request body: %s.", err)
			return
		}

		tokenResponse, err := userCtrl.CreateAccessToken(ctx, session, userUID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusCreated, tokenResponse)
	}
}
