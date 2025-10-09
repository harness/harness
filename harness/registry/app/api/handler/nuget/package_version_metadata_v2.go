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

package nuget

import (
	"encoding/xml"
	"fmt"
	"net/http"

	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) GetPackageVersionMetadataV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*nugettype.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch info from context")}, w)
		return
	}
	response := h.controller.GetPackageVersionMetadataV2(r.Context(), *info)

	if response.GetError() != nil {
		h.HandleError(r.Context(), w, response.GetError())
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		h.HandleErrors(r.Context(), []error{err}, w)
		return
	}

	err = xml.NewEncoder(w).Encode(response.FeedEntryResponse)
	if err != nil {
		h.HandleErrors(r.Context(), []error{err}, w)
		return
	}
}
