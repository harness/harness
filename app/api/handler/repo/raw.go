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
	"fmt"
	"net/http"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"

	"github.com/rs/zerolog/log"
)

// HandleRaw returns the raw content of a file.
func HandleRaw(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		gitRef := request.GetGitRefFromQueryOrDefault(r, "")
		path := request.GetOptionalRemainderFromPath(r)

		resp, err := repoCtrl.Raw(ctx, session, repoRef, gitRef, path)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		defer func() {
			if err := resp.Data.Close(); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader.")
			}
		}()

		ifNoneMatch, ok := request.GetIfNoneMatchFromHeader(r)
		if ok && ifNoneMatch == resp.SHA.String() {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Add("Content-Length", fmt.Sprint(resp.Size))
		w.Header().Add(request.HeaderETag, resp.SHA.String())
		render.Reader(ctx, w, http.StatusOK, resp.Data)
	}
}
