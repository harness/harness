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
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/errors"
	gittypes "github.com/harness/gitness/git/api"
)

// HandleDiff returns the diff between two commits, branches or tags.
func HandleDiff(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		path := request.GetOptionalRemainderFromPath(r)

		files := gittypes.FileDiffRequests{}
		switch r.Method {
		case http.MethodPost:
			if err = json.NewDecoder(r.Body).Decode(&files); err != nil && !errors.Is(err, io.EOF) {
				render.TranslatedUserError(ctx, w, err)
				return
			}
		case http.MethodGet:
			// TBD: this will be removed in future because of URL limit in browser to 2048 chars.
			files = request.GetFileDiffFromQuery(r)
		}

		ignoreWhitespace, err := request.QueryParamAsBoolOrDefault(r, request.QueryParamIgnoreWhitespace, false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		if strings.HasPrefix(r.Header.Get("Accept"), "text/plain") {
			err := repoCtrl.RawDiff(
				ctx,
				w,
				session,
				repoRef,
				path,
				ignoreWhitespace,
				files...,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusOK)
			}
			return
		}

		includePatch, err := request.QueryParamAsBoolOrDefault(r, request.QueryParamIncludePatch, false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		stream, err := repoCtrl.Diff(
			ctx,
			session,
			repoRef,
			path,
			includePatch,
			ignoreWhitespace,
			files...,
		)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
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
			render.TranslatedUserError(ctx, w, err)
			return
		}

		commitSHA, err := request.GetCommitSHAFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		ignoreWhitespace, err := request.QueryParamAsBoolOrDefault(r, request.QueryParamIgnoreWhitespace, false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		err = repoCtrl.CommitDiff(
			ctx,
			session,
			repoRef,
			commitSHA,
			ignoreWhitespace,
			w,
		)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
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
			render.TranslatedUserError(ctx, w, err)
			return
		}

		path := request.GetOptionalRemainderFromPath(r)

		ignoreWhitespace, err := request.QueryParamAsBoolOrDefault(r, request.QueryParamIgnoreWhitespace, false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		output, err := repoCtrl.DiffStats(
			ctx,
			session,
			repoRef,
			path,
			ignoreWhitespace,
		)
		if uErr := gittypes.AsUnrelatedHistoriesError(err); uErr != nil {
			render.JSON(w, http.StatusOK, &usererror.Error{
				Message: uErr.Error(),
				Values:  uErr.Map(),
			})
			return
		}
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, output)
	}
}
