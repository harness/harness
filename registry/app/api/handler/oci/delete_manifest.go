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
	"net/http"

	"github.com/harness/gitness/registry/app/pkg/commons"

	"github.com/rs/zerolog/log"
)

// PutManifest validates and stores a manifest in the registry.
func (h *Handler) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.GetRegistryInfo(r, false)
	if err != nil {
		handleErrors(r.Context(), []error{err}, w)
		return
	}
	length := r.ContentLength
	if length > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, length)
	}
	errs, headers := h.Controller.DeleteManifest(r.Context(), info)

	if !commons.IsEmpty(errs) {
		log.Ctx(ctx).Error().Msgf("DeleteManifest: %v", errs)
	}

	if headers != nil {
		headers.WriteToResponse(w)
	}
	handleErrors(ctx, errs, w)
}
