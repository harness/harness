//  Copyright 2023 Harness, Inc.
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

package pypi

import (
	"net/http"

	"github.com/harness/gitness/registry/app/pkg/commons"

	"github.com/go-chi/chi/v5"
)

func (h *handler) DownloadPackageFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.GetArtifactInfo(r)
	if !commons.IsEmptyError(err) {
		h.HandleErrors(r.Context(), err, w)
		return
	}

	image := chi.URLParam(r, "image")
	filename := chi.URLParam(r, "filename")
	version := chi.URLParam(r, "version")
	headers, fileReader, redirectURL, err := h.controller.DownloadPackageFile(ctx, info, image,
		version, filename)
	if commons.IsEmptyError(err) {
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		if redirectURL != "" {
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		h.ServeContent(w, r, fileReader, filename)
		headers.WriteToResponse(w)
		return
	}
	h.HandleErrors(r.Context(), err, w)
}
