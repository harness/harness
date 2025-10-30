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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) SearchPackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*nugettype.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch info from context")}, w)
		return
	}
	searchTerm := r.URL.Query().Get("q")
	offset, err := strconv.Atoi(r.URL.Query().Get("skip"))
	if err != nil {
		offset = 0
	}
	limit, err2 := strconv.Atoi(r.URL.Query().Get("take"))
	if err2 != nil {
		limit = 20
	}
	response := h.controller.SearchPackage(r.Context(), *info, searchTerm, limit, offset)

	if response.GetError() != nil {
		h.HandleError(r.Context(), w, response.GetError())
		return
	}
	err3 := json.NewEncoder(w).Encode(response.SearchResponse)
	if err3 != nil {
		h.HandleErrors(r.Context(), []error{err3}, w)
		return
	}
}
