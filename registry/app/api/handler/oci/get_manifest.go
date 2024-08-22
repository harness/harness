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

package oci

import (
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/docker"

	"github.com/rs/zerolog/log"
)

// GetManifest fetches the image manifest from the storage backend, if it exists.
func (h *Handler) GetManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.getRegistryInfo(r, true)
	if err != nil {
		handleErrors(ctx, errcode.Errors{err}, w)
		return
	}

	result := h.Controller.PullManifest(
		ctx,
		info,
		r.Header[commons.HeaderAccept],
		r.Header[commons.HeaderIfNoneMatch],
	)
	if commons.IsEmpty(result.GetErrors()) {
		response, ok := result.(*docker.GetManifestResponse)
		if !ok {
			log.Ctx(ctx).Error().Msg("Failed to cast result to GetManifestResponse")
			return
		}
		response.ResponseHeaders.WriteToResponse(w)
		_, bytes, _ := response.Manifest.Payload()
		if _, err := w.Write(bytes); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to write response")
			response.ResponseHeaders.Code = http.StatusInternalServerError
		}
		return
	}
	handleErrors(ctx, result.GetErrors(), w)
}
