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
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/docker"

	"github.com/rs/zerolog/log"
)

func (h *Handler) GetBlob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.getRegistryInfo(r, true)
	if err != nil {
		handleErrors(r.Context(), []error{err}, w)
		return
	}
	result := h.Controller.GetBlob(ctx, info)

	response, ok := result.(*docker.GetBlobResponse)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to cast result to GetBlobResponse")
		handleErrors(ctx, []error{errors.New("failed to cast result to GetBlobResponse")}, w)
		return
	}
	defer func() {
		if response.Body != nil {
			response.Body.Close()
		}
		if response.ReadCloser != nil {
			response.ReadCloser.Close()
		}
	}()

	if commons.IsEmpty(response.GetErrors()) {
		if !commons.IsEmpty(response.RedirectURL) {
			http.Redirect(w, r, response.RedirectURL, http.StatusTemporaryRedirect)
			return
		}
		response.ResponseHeaders.WriteHeadersToResponse(w)
		if r.Method == http.MethodHead {
			return
		}

		h.serveContent(w, r, response, info)
		response.ResponseHeaders.WriteToResponse(w)
	}

	handleErrors(r.Context(), response.GetErrors(), w)
}

func (h *Handler) serveContent(
	w http.ResponseWriter, r *http.Request, response *docker.GetBlobResponse, info pkg.RegistryInfo,
) {
	if response.Body != nil {
		http.ServeContent(w, r, info.Digest, time.Time{}, response.Body)
	} else {
		// Use io.CopyN to avoid out of memory when pulling big blob
		written, err2 := io.CopyN(w, response.ReadCloser, response.Size)
		if err2 != nil {
			response.Errors = append(response.Errors, errors.New("error copying blob to response"))
			log.Ctx(r.Context()).Error().Msg("error copying blob to response:")
		}
		if written != response.Size {
			response.Errors = append(
				response.Errors,
				fmt.Errorf(fmt.Sprintf("The size mismatch, actual:%d, expected: %d", written, response.Size)),
			)
		}
	}
}
