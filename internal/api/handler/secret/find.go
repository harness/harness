// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package secret

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/secret"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/paths"
)

// HandleFind finds a secret from the database.
func HandleFind(secretCtrl *secret.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		secretRef, err := request.GetSecretRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, secretUID, err := paths.DisectLeaf(secretRef)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		secret, err := secretCtrl.Find(ctx, session, spaceRef, secretUID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, secret.CopyWithoutData())
	}
}
