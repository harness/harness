// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

/*
 * Deletes a given path.
 */
func HandleDeletePath(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		pathID, err := request.GetPathIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		err = repoCtrl.DeletePath(ctx, session, repoRef, pathID)
		if err != nil {
			render.TranslatedUserError(w, err)
		}

		render.DeleteSuccessful(w)
	}
}
