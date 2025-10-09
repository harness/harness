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

	"github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

// PutFile handles file upload requests.
func (h *Handler) PutFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	artifactInfo := request.ArtifactInfoFrom(ctx)
	info, ok := artifactInfo.(generic.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get generic artifact info from context")
		h.HandleError(ctx, w, fmt.Errorf("failed to fetch info from context"))
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload file
	response := h.Controller.PutFile(ctx, info, r.Body, contentType)
	if response.GetError() != nil {
		log.Ctx(ctx).Error().Err(response.GetError()).Msg("failed to upload file")
		h.HandleError(ctx, w, response.GetError())
		return
	}

	// Set response headers
	if response.ResponseHeaders != nil {
		response.ResponseHeaders.WriteToResponse(w)
	}

	w.WriteHeader(http.StatusCreated)
}
