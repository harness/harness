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
	"fmt"
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg/commons"
)

func (h *handler) UploadPackageFile(w http.ResponseWriter, r *http.Request) {
	file, fileHeader, err := r.FormFile("content")
	if err != nil {
		h.HandleErrors(r.Context(), errcode.ErrCodeInvalidRequest.WithMessage(fmt.Sprintf("failed to parse file: %s, "+
			"please provide correct file path ", err.Error())), w)
		return
	}

	defer file.Close()

	info, err := h.getPackageArtifactInfo(r)
	if err != nil {
		h.HandleErrors(r.Context(),
			errcode.ErrCodeInvalidRequest.WithMessage("failed to get artifact info "+err.Error()), w)
		return
	}

	headers, sha256, err2 := h.controller.UploadPackageFile(r.Context(), info, file, fileHeader)

	if commons.IsEmptyError(err2) {
		headers.WriteToResponse(w)
		_, err := w.Write([]byte(fmt.Sprintf("Pushed.\nSha256: %s", sha256)))
		if err != nil {
			h.HandleErrors(r.Context(), errcode.ErrCodeUnknown.WithDetail(err2), w)
			return
		}
	}
	h.HandleErrors(r.Context(), err2, w)
}
