// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/githook"
	githookcontroller "github.com/harness/gitness/internal/api/controller/githook"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleUpdate returns a handler function that handles update git hooks.
func HandleUpdate(githookCtrl *githookcontroller.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoID, err := request.GetRepoIDFromQuery(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		principalID, err := request.GetPrincipalIDFromQuery(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		in := new(githook.UpdateInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		out, err := githookCtrl.Update(ctx, session, repoID, principalID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, out)
	}
}
