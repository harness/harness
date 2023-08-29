// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package template

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/template"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/paths"
)

// HandleFind finds a template from the database.
func HandleFind(templateCtrl *template.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		templateRef, err := request.GetTemplateRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, templateUID, err := paths.DisectLeaf(templateRef)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		template, err := templateCtrl.Find(ctx, session, spaceRef, templateUID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, template)
	}
}
