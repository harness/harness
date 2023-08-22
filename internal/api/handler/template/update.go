// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package template

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/template"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/paths"
)

func HandleUpdate(templateCtrl *template.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := new(template.UpdateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		templateRef, err := request.GetTemplateRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, templateUID, err := paths.DisectLeaf(templateRef)
		if err != nil {
			render.TranslatedUserError(w, err)
		}

		template, err := templateCtrl.Update(ctx, session, spaceRef, templateUID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, template)
	}
}
