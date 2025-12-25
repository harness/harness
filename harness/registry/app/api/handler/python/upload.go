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

package python

import (
	"fmt"
	"net/http"

	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/request"
)

func (h *handler) UploadPackageFile(w http.ResponseWriter, r *http.Request) {
	file, fileHeader, err := r.FormFile("content")

	if err != nil {
		h.HandleError(r.Context(), w, err)
		return
	}

	defer file.Close()

	contextInfo := request.ArtifactInfoFrom(r.Context())
	info, ok := contextInfo.(*pythontype.ArtifactInfo)
	if !ok {
		h.HandleError(r.Context(), w, fmt.Errorf("failed to fetch info from context"))
		return
	}

	// TODO: Can we extract this out to ArtifactInfoProvider
	if info.Filename == "" {
		info.Filename = fileHeader.Filename
		request.WithArtifactInfo(r.Context(), info)
	}

	response := h.controller.UploadPackageFile(r.Context(), *info, file, fileHeader)

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
