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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/request"
)

func (h *handler) UploadPackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*cargotype.ArtifactInfo)
	if !ok {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to fetch info from context"))
		return
	}

	// parse metadata, and crate file from payload
	metadata, fileReader, err := h.parseDataFromPayload(r.Body)
	if err != nil {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to parse data from payload: %w", err))
		return
	}

	info.Image = metadata.Name
	info.Version = metadata.Version

	response, err := h.controller.UploadPackage(ctx, info, metadata, fileReader)
	if err != nil {
		h.handleCargoPackageAPIError(w, r, fmt.Errorf("failed to upload package: %w", err))
		return
	}

	// Final response
	response.ResponseHeaders.WriteToResponse(w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.handleCargoPackageAPIError(w, r,
			fmt.Errorf("error occurred during sending response for upload package for cargo package: %w", err),
		)
	}
}

func (h *handler) parseDataFromPayload(
	fileReader io.ReadCloser,
) (*cargometadata.VersionMetadata, io.ReadCloser, error) {
	// Step 1: Read first 4 bytes to get JSON length
	header := make([]byte, 4)
	if _, err := io.ReadFull(fileReader, header); err != nil {
		return nil, nil, fmt.Errorf("failed to read JSON length: %w", err)
	}
	jsonLen := binary.LittleEndian.Uint32(header)
	// Step 2: Read the JSON metadata
	jsonBuf := make([]byte, jsonLen)
	if _, err := io.ReadFull(fileReader, jsonBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to read JSON metadata: %w", err)
	}

	var metadata *cargometadata.VersionMetadata
	if err := json.Unmarshal(jsonBuf, &metadata); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON: %w", err)
	}
	// 3. Read 4 bytes: crate length
	var crateLenBuf [4]byte
	if _, err := io.ReadFull(fileReader, crateLenBuf[:]); err != nil {
		return nil, nil, fmt.Errorf("failed to read crate length: %w", err)
	}
	crateLen := binary.LittleEndian.Uint32(crateLenBuf[:])
	crateReader := io.LimitReader(fileReader, int64(crateLen))
	metadata.Yanked = false // Ensure Yanked is false for new uploads
	return metadata, io.NopCloser(crateReader), nil
}
