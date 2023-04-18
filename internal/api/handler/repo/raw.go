// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleRaw returns the raw content of a file.
func HandleRaw(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		gitRef := request.GetGitRefFromQueryOrDefault(r, "")
		path := request.GetOptionalRemainderFromPath(r)

		dataReader, dataLength, err := repoCtrl.Raw(ctx, session, repoRef, gitRef, path)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		w.Header().Add("Content-Length", fmt.Sprint(dataLength))

		render.Reader(ctx, w, http.StatusOK, dataReader)
	}
}
