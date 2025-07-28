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

package gopackage

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/gopackage/utils"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) DownloadPackageFile(
	w http.ResponseWriter, r *http.Request,
) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*gopackagetype.ArtifactInfo)
	if !ok {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	path := r.PathValue("*")
	image, version, filename, err := utils.GetArtifactInfoFromURL(path)
	if err != nil {
		http.Error(w,
			fmt.Sprintf("image and version not found in path %s: %s", path, err.Error()),
			http.StatusNotFound, // go client need 404 for incorrect path
		)
		return
	}

	info.Image = image
	info.Version = version
	info.FileName = filename

	response := h.controller.DownloadPackageFile(ctx, info)
	if response == nil {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to get response from controller"))
		return
	}

	defer func() {
		if response.Body != nil {
			err := response.Body.Close()
			if err != nil {
				log.Ctx(ctx).Error().Msgf("Failed to close body: %v", err)
			}
		}
		if response.ReadCloser != nil {
			err := response.ReadCloser.Close()
			if err != nil {
				log.Ctx(ctx).Error().Msgf("Failed to close read closer: %v", err)
			}
		}
	}()

	if response.GetError() != nil {
		h.handleGoPackageAPIError(w, r, response.GetError())
		return
	}

	if response.RedirectURL != "" {
		http.Redirect(w, r, response.RedirectURL, http.StatusTemporaryRedirect)
		return
	}

	err = commons.ServeContent(w, r, response.Body, info.FileName, response.ReadCloser)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to serve content: %v", err)
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to serve content: %w", err))
		return
	}
	response.ResponseHeaders.WriteToResponse(w)
}
