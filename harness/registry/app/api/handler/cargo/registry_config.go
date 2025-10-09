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

package cargo

import (
	"encoding/json"
	"fmt"
	"net/http"

	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/request"
)

func (h *handler) GetRegistryConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*cargotype.ArtifactInfo)
	if !ok {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	response, err := h.controller.GetRegistryConfig(ctx, info)
	if err != nil {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to fetch registry config %w", err))
		return
	}

	response.ResponseHeaders.WriteHeadersToResponse(w)
	w.Header().Set("Content-Type", "text/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(response.Config)
	if err != nil {
		h.handleCargoPackageAPIError(w, r,
			fmt.Errorf("error occurred during sending response for get registry config for cargo package: %w", err),
		)
	}
}
