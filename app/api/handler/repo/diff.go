// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repo

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
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
			err := repoCtrl.RawDiff(ctx, session, repoRef, path, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusOK)
			}
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

// HandleCommitDiff returns the diff between two commits, branches or tags.
func HandleCommitDiff(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		commitSHA, err := request.GetCommitSHAFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		err = repoCtrl.CommitDiff(ctx, session, repoRef, commitSHA, w)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
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
