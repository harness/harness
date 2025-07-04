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

	"github.com/oapi-codegen/runtime"
)

func (h *handler) SearchPackage(
	w http.ResponseWriter, r *http.Request,
) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*cargotype.ArtifactInfo)
	if !ok {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	requestInfo, err := h.getSearchPackageParams(r)
	if err != nil {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to get search package params: %w", err))
		return
	}
	response, err := h.controller.SearchPackage(ctx, info, requestInfo)
	if err != nil {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to get response from controller: %w", err))
		return
	}

	response.ResponseHeaders.WriteHeadersToResponse(w)
	w.Header().Set("Content-Type", "text/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.handleCargoPackageAPIError(w, r,
			fmt.Errorf("error occurred during sending response for search cargo package: %w", err),
		)
	}
}

func (h *handler) getSearchPackageParams(r *http.Request) (*cargotype.SearchPackageRequestParams, error) {
	var params cargotype.SearchPackageRequestParams
	err := runtime.BindQueryParameter("form", true, false, "q", r.URL.Query(), &params.SearchTerm)
	if err != nil {
		return nil, fmt.Errorf("invalid format for parameter %s: %w", "q", err)
	}

	err = runtime.BindQueryParameter("form", true, false, "per_page", r.URL.Query(), &params.Size)
	if err != nil {
		return nil, fmt.Errorf("invalid format for parameter %s: %w", "per_page", err)
	}

	return &params, nil
}
