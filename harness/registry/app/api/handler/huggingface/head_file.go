// Copyright 2023 Harness, Inc.
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

package huggingface

import (
	"fmt"
	"net/http"
	"net/url"

	huggingfacetype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) HeadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*huggingfacetype.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch info from context")}, w)
		return
	}
	filePath := r.PathValue("*")
	decodedFilePath, err := url.PathUnescape(filePath)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to decode file path: %v", err)
		h.HandleError(r.Context(), w, err)
		return
	}
	response := h.controller.HeadFile(r.Context(), *info, decodedFilePath)

	if response.GetError() != nil {
		h.HandleError(r.Context(), w, response.GetError())
		return
	}
	response.ResponseHeaders.WriteToResponse(w)
}
