// Copyright 2026 Harness, Inc.
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
	"strings"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/services/codeowners"
)

// contentTypeTextPlainUTF8 is the content type the CODEOWNERS file is served with. It is always
// text, so unlike the raw file endpoint there is no need to sniff the content type.
const contentTypeTextPlainUTF8 = "text/plain; charset=utf-8"

// HandleCodeOwnersRaw returns the CODEOWNERS file of the repository as committed, as plain text.
func HandleCodeOwnersRaw(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		gitRef := request.GetGitRefFromQueryOrDefault(r, "")
		file, err := repoCtrl.CodeOwnersRaw(ctx, session, repoRef, gitRef)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		writeCodeOwnersRawResponse(w, r, file)
	}
}

// writeCodeOwnersRawResponse is separate from HandleCodeOwnersRaw so that the headers can be
// tested without standing up a repo controller.
func writeCodeOwnersRawResponse(w http.ResponseWriter, r *http.Request, file *codeowners.File) {
	// The file is committed by repo users and served from the app origin. Set the sandbox headers
	// before the 304 below so that no response can reach the browser without them.
	render.UserContentSecurityHeaders(w)

	// RFC 9110 requires an entity-tag to be DQUOTE-delimited, so that proxies which validate the
	// header don't drop it. The If-None-Match comparison below has to match on the same form,
	// otherwise a client echoing our own ETag back would never get a 304.
	etag := `"` + file.SHA + `"`
	w.Header().Set(request.HeaderETag, etag)

	if matchesETag(r, file.SHA) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", contentTypeTextPlainUTF8)
	w.Header().Set("Content-Length", fmt.Sprint(len(file.Content)))

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(file.Content))
}

// matchesETag reports whether the request's If-None-Match header refers to the given blob SHA.
// Weak prefixes and quoting are normalized away, and a bare SHA is accepted so that clients
// written against the unquoted form keep revalidating successfully.
func matchesETag(r *http.Request, blobSHA string) bool {
	header, ok := request.GetIfNoneMatchFromHeader(r)
	if !ok || blobSHA == "" {
		return false
	}

	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		candidate = strings.TrimPrefix(candidate, "W/")
		if strings.Trim(candidate, `"`) == blobSHA {
			return true
		}
	}

	return false
}
