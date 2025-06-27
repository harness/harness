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

package generic

import (
	"net/http"
	"time"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
)

func (h *Handler) PullArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.GetGenericArtifactInfo(r)
	if !commons.IsEmptyError(err) {
		handleErrors(r.Context(), err, w)
		return
	}
	if info.FileName == "" {
		info.FileName = r.FormValue("filename")
	}

	headers, fileReader, redirectURL, err := h.Controller.PullArtifact(ctx, info)
	if commons.IsEmptyError(err) {
		w.Header().Set("Content-Disposition", "attachment; filename="+info.FileName)
		if redirectURL != "" {
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		h.serveContent(w, r, fileReader, info)
		headers.WriteToResponse(w)
		return
	}
	handleErrors(r.Context(), err, w)
}

func (h *Handler) serveContent(
	w http.ResponseWriter, r *http.Request, fileReader *storage.FileReader, info pkg.GenericArtifactInfo,
) {
	if fileReader != nil {
		http.ServeContent(w, r, info.FileName, time.Time{}, fileReader)
	}
}
