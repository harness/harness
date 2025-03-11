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

package lfs

import (
	"errors"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/lfs"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/url"
)

func HandleLFSDownload(controller *lfs.Controller, urlProvider url.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		oid, err := request.GetObjectIDFromQuery(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		file, err := controller.Download(ctx, session, repoRef, oid)
		if errors.Is(err, apiauth.ErrNotAuthorized) && auth.IsAnonymousSession(session) {
			render.GitBasicAuth(ctx, w, urlProvider)
			return
		}
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		defer file.Close()
		// apply max byte size
		render.Reader(ctx, w, http.StatusOK, file)
	}
}
