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
	"strconv"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/url"
)

// HandleGitRedirect redirects from the vanilla git clone URL to the repo UI page.
func HandleGitRedirect(urlProvider url.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		// Explicitly return error in case the user is trying to use the repoID for redirect.
		if _, err := strconv.ParseInt(repoRef, 10, 64); err == nil {
			render.BadRequestf(ctx, w, "Endpoint only supports repo path.")
			return
		}

		// Always use the raw, user-provided path to generate the redirect URL.
		// NOTE:
		//   Technically, we could find the repo first and use repo.Path.
		//   However, the auth cookie isn't available in case of custom git domains, and thus the auth would fail.
		repoURL := urlProvider.GenerateUIRepoURL(ctx, repoRef)

		http.Redirect(
			w,
			r,
			repoURL,
			http.StatusMovedPermanently,
		)
	}
}
