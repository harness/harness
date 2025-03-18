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

	"github.com/harness/gitness/registry/app/pkg/commons"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) DownloadPackageFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*pythontype.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get python artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch python artifact info from context")}, w)
		return
	}

	response := h.controller.DownloadPackageFile(ctx, *info)

	defer func() {
		if response.Body != nil {
			err := response.Body.Close()
			if err != nil {
				log.Ctx(r.Context()).Error().Msgf("Failed to close body: %v", err)
			}
		}
	}()

	if !commons.IsEmpty(response.GetErrors()) {
		h.HandleErrors(r.Context(), response.GetErrors(), w)
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+info.Filename)
	if response.RedirectURL != "" {
		http.Redirect(w, r, response.RedirectURL, http.StatusTemporaryRedirect)
		return
	}
	h.ServeContent(w, r, response.Body, info.Filename)
	response.ResponseHeaders.WriteToResponse(w)
}
