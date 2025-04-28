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
	"fmt"
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg/commons"
)

func (h *Handler) PushArtifact(w http.ResponseWriter, r *http.Request) {
	info, err := h.GetGenericArtifactInfo(r)
	if !commons.IsEmptyError(err) {
		handleErrors(r.Context(), err, w)
		return
	}

	file, _, err1 := r.FormFile("file")
	if err1 != nil {
		handleErrors(r.Context(),
			errcode.ErrCodeInvalidRequest.WithMessage(fmt.Sprintf("failed to parse file: %s, "+
				"please provide correct file path ", err.Message)), w)
		return
	}
	ctx := r.Context()
	defer file.Close()
	headers, sha256, err := h.Controller.UploadArtifact(ctx, info, file)
	if commons.IsEmptyError(err) {
		headers.WriteToResponse(w)
		_, err := w.Write([]byte(fmt.Sprintf("Pushed.\nSha256: %s", sha256)))
		if err != nil {
			handleErrors(r.Context(), errcode.ErrCodeUnknown.WithDetail(err), w)
			return
		}
	}
	handleErrors(r.Context(), err, w)
}
