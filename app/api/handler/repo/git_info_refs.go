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
	"context"
	"fmt"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
)

// HandleGitInfoRefs handles the info refs part of git's smart http protocol.
func HandleGitInfoRefs(repoCtrl *repo.Controller, urlProvider url.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			pktError(ctx, w, err)
			return
		}

		gitProtocol := request.GetGitProtocolFromHeadersOrDefault(r, "")
		service, err := request.GetGitServiceTypeFromQuery(r)
		if err != nil {
			pktError(ctx, w, err)
			return
		}

		// Clients MUST NOT reuse or revalidate a cached response.
		// Servers MUST include sufficient Cache-Control headers to prevent caching of the response.
		// https://git-scm.com/docs/http-protocol
		render.NoCache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))

		err = repoCtrl.GitInfoRefs(ctx, session, repoRef, service, gitProtocol, w)
		if errors.Is(err, apiauth.ErrForbidden) && auth.IsAnonymousSession(session) {
			render.GitBasicAuth(ctx, w, urlProvider)
			return
		}
		if err != nil {
			pktError(ctx, w, err)
			return
		}
	}
}

func pktError(ctx context.Context, w http.ResponseWriter, err error) {
	terr := usererror.Translate(ctx, err)
	w.WriteHeader(terr.Status)
	api.PktError(w, terr)
}
