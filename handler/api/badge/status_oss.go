// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build oss

package badge

import (
	"io"
	"net/http"
	"time"

	"github.com/drone/drone/core"

	"github.com/go-chi/chi"
)

// Handler returns a http.HandlerFunc that responds with the build status in svg form.
func Handler(repositoryStore core.RepositoryStore, buildStore core.BuildStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Type", "image/svg+xml")

		owner := chi.URLParam(r, "owner")
		name := chi.URLParam(r, "name")
		ref := r.FormValue("ref")

		repo, err := repositoryStore.FindName(r.Context(), owner, name)
		if err != nil {
			io.WriteString(w, badgeNone)
			return
		}

		if ref == "" {
			ref = "refs/heads/" + repo.Branch
		}

		build, err := buildStore.FindRef(r.Context(), repo.ID, ref)
		if err != nil {
			io.WriteString(w, badgeNone)
			return
		}

		switch build.Status {
		case core.StatusPassing:
			io.WriteString(w, badgeSuccess)
		case core.StatusPending, core.StatusRunning, core.StatusWaiting:
			io.WriteString(w, badgeStarted)
		case core.StatusError:
			io.WriteString(w, badgeError)
		default:
			io.WriteString(w, badgeFailure)
		}
	}
}
