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

func (h *handler) UnYankVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*cargotype.ArtifactInfo)
	if !ok {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	response, err := h.controller.UpdateYank(ctx, info, false)
	if err != nil {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to unyank version: %w", err))
		return
	}

	// Final response
	response.ResponseHeaders.Headers["Content-Type"] = "application/json"
	response.ResponseHeaders.WriteToResponse(w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.handleCargoPackageAPIError(w, r,
			fmt.Errorf("error occurred during sending response for unyank version for cargo package: %w", err),
		)
	}
}
