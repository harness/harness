// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

/*
 * Deletes a given path.
 */
func HandleDeletePath(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		spaceRef, err := request.GetSpaceRef(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		pathID, err := request.GetPathID(r)
		if err != nil {
			render.BadRequest(w)
			return
		}

		err = spaceCtrl.DeletePath(ctx, session, spaceRef, pathID)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
