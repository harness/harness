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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/registry/app/pkg/gopackage/utils"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/request"
)

func (h *handler) UploadPackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*gopackagetype.ArtifactInfo)
	if !ok {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	// parse metadata, info, mod and zip files
	infoBytes, modBytes, zipRC, err := h.parseDataFromPayload(r)
	if err != nil {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to parse data from payload: %w", err))
		return
	}

	// get package metadata from info file
	metadata, err := utils.GetPackageMetadataFromInfoFile(infoBytes)
	if err != nil {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to get package metadata from info file: %w", err))
		return
	}

	// get module name from mod file
	moduleName, err := utils.GetModuleNameFromModFile(bytes.NewReader(modBytes.Bytes()))
	if err != nil {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to get module name from mod file: %w", err))
		return
	}

	metadata.Name = moduleName

	// update artifact info with required data
	info.Metadata = metadata
	info.Version = metadata.Version
	info.Image = metadata.Name

	modRC := io.NopCloser(bytes.NewReader(modBytes.Bytes()))
	defer modRC.Close()
	defer zipRC.Close()

	// upload package
	response := h.controller.UploadPackage(ctx, info, modRC, zipRC)
	if response.GetError() != nil {
		h.handleGoPackageAPIError(w, r, fmt.Errorf("failed to upload package: %w", response.GetError()))
		return
	}

	// Final response
	w.Header().Set("Content-Type", "application/json")
	response.ResponseHeaders.WriteToResponse(w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.handleGoPackageAPIError(w, r,
			fmt.Errorf("error occurred during sending response for upload package for go package: %w", err),
		)
	}
}
