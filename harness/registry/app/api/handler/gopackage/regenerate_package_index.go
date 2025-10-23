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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/request"
)

func (h *handler) RegeneratePackageIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*gopackagetype.ArtifactInfo)
	if !ok {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	info.Image = strings.Trim(r.URL.Query().Get("image"), "'")
	if info.Image == "" {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("image is required"))
		return
	}

	// regenerate package index
	response := h.controller.RegeneratePackageIndex(ctx, info)
	if response.GetError() != nil {
		h.handleGoPackageAPIError(
			w, r, fmt.Errorf("failed to regenerate package index: %w", response.GetError()),
		)
		return
	}

	// Final response
	w.Header().Set("Content-Type", "application/json")
	response.ResponseHeaders.WriteToResponse(w)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		h.handleGoPackageAPIError(w, r,
			fmt.Errorf("error occurred during sending response for regenerate package index for go package: %w", err),
		)
	}
}
