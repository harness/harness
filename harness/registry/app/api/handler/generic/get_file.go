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

	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

// GetFile handles file download and metadata retrieval requests.
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	info := request.ArtifactInfoFrom(ctx)
	artifactInfo, ok := info.(generic.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get generic artifact info from context")
		h.HandleError(ctx, w, fmt.Errorf("failed to fetch info from context"))
		return
	}

	response := h.Controller.DownloadFile(ctx, artifactInfo,
		artifactInfo.FilePath)

	if response == nil {
		h.HandleErrors(ctx, []error{fmt.Errorf("failed to get response from controller")}, w)
		return
	}

	defer func() {
		if response.Body != nil {
			err := response.Body.Close()
			if err != nil {
				log.Ctx(ctx).Warn().Msgf("Failed to close body: %v", err)
			}
		}

		if response.ReadCloser != nil {
			err := response.ReadCloser.Close()
			if err != nil {
				log.Ctx(ctx).Warn().Msgf("Failed to close read closer body: %v", err)
			}
		}
	}()

	if response.GetError() != nil {
		h.HandleError(ctx, w, response.GetError())
		return
	}

	if response.RedirectURL != "" {
		http.Redirect(w, r, response.RedirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Set content disposition for download
	if info.GetFileName() != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.GetFileName()))
	}
	err := commons.ServeContent(w, r, response.Body, info.GetFileName(), response.ReadCloser)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to serve content: %v", err)
		h.HandleError(ctx, w, err)
		return
	}
	response.ResponseHeaders.WriteToResponse(w)
}
