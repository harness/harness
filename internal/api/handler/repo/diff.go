// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleDiff returns the diff between two commits, branches or tags.
func HandleDiff(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		path := request.GetOptionalRemainderFromPath(r)

		if strings.HasPrefix(r.Header.Get("Accept"), "text/plain") {
			// error checking is intentionally skipped because we dont want to send errors as text/plain
			_ = repoCtrl.RawDiff(ctx, session, repoRef, path, w)
			return
		}

		_, includePatch := request.QueryParam(r, "include_patch")
		stream, err := repoCtrl.Diff(ctx, session, repoRef, path, includePatch)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSONArrayDynamic(ctx, w, stream)
	}
}

// HandleDiffStats how diff statistics of two commits, branches or tags.
func HandleDiffStats(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		path := request.GetOptionalRemainderFromPath(r)

		output, err := repoCtrl.DiffStats(ctx, session, repoRef, path)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, output)
	}
}
