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

package maven

import (
	"net/http"
	"time"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven"
	"github.com/harness/gitness/registry/app/storage"

	"github.com/rs/zerolog/log"
)

func (h *Handler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.GetArtifactInfo(r, true)
	if err != nil {
		handleErrors(ctx, []error{err}, w)
		return
	}
	result := h.Controller.GetArtifact(
		ctx,
		info,
	)
	if !commons.IsEmpty(result.GetErrors()) {
		handleErrors(ctx, result.GetErrors(), w)
		return
	}
	response, ok := result.(*maven.GetArtifactResponse)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to cast result to GetArtifactResponse")
		return
	}

	defer func() {
		if response.Body != nil {
			err := response.Body.Close()
			if err != nil {
				log.Ctx(ctx).Error().Msgf("Failed to close body: %v", err)
			}
		}
	}()

	if !commons.IsEmpty(response.RedirectURL) {
		http.Redirect(w, r, response.RedirectURL, http.StatusTemporaryRedirect)
		return
	}
	h.serveContent(w, r, response.Body, info)
	response.ResponseHeaders.WriteToResponse(w)
}

func (h *Handler) serveContent(
	w http.ResponseWriter, r *http.Request, fileReader *storage.FileReader, info pkg.MavenArtifactInfo,
) {
	if fileReader != nil {
		http.ServeContent(w, r, info.FileName, time.Time{}, fileReader)
	}
}
