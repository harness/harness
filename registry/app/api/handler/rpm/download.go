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
	"net/url"

	"github.com/harness/gitness/registry/app/pkg/commons"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) DownloadPackageFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*rpmtype.ArtifactInfo)
	if !ok {
		h.HandleErrors(ctx, []error{fmt.Errorf("failed to fetch info from context")}, w)
		return
	}
	info.Image = r.PathValue("name")
	info.Arch = r.PathValue("architecture")
	version, err := url.PathUnescape(r.PathValue("version"))
	if err != nil {
		h.HandleErrors(ctx, []error{fmt.Errorf("failed to decode version: %s", r.PathValue("version"))}, w)
		return
	}
	info.Version = version + "." + info.Arch
	info.FileName = r.PathValue("file")
	info.PackagePath = r.PathValue("*")
	response := h.controller.DownloadPackageFile(ctx, *info)
	if response == nil {
		h.HandleErrors(ctx, []error{fmt.Errorf("failed to get response from controller")}, w)
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
		h.HandleError(ctx, w, response.GetError())
		return
	}

	if response.RedirectURL != "" {
		http.Redirect(w, r, response.RedirectURL, http.StatusTemporaryRedirect)
		return
	}

	err = commons.ServeContent(w, r, response.Body, info.FileName, response.ReadCloser)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to serve content: %v", err)
		h.HandleError(ctx, w, err)
		return
	}
	response.ResponseHeaders.WriteToResponse(w)
}
