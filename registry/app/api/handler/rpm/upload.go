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

package rpm

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/request"
)

const formFileKey = "file"

func (h *handler) UploadPackageFile(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile(formFileKey)
	if err != nil {
		h.HandleErrors2(r.Context(), errcode.ErrCodeInvalidRequest.WithMessage(fmt.Sprintf("failed to parse file: %s, "+
			"please provide correct file path ", err.Error())), w)
		return
	}
	defer file.Close()

	contextInfo := request.ArtifactInfoFrom(r.Context())
	info, ok := contextInfo.(*rpmtype.ArtifactInfo)
	if !ok {
		h.HandleErrors2(r.Context(), errcode.ErrCodeInvalidRequest.WithMessage("failed to fetch info from context"), w)
		return
	}

	response := h.controller.UploadPackageFile(r.Context(), *info, file)
	if response.GetError() != nil {
		h.HandleError(r.Context(), w, response.GetError())
		return
	}

	response.ResponseHeaders.WriteToResponse(w)
	_, err = w.Write([]byte(fmt.Sprintf("Pushed.\nSha256: %s", response.Sha256)))
	if err != nil {
		h.HandleError(r.Context(), w, err)
		return
	}
}
