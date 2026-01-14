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

package maven

import (
	"bytes"
	"net/http"

	"github.com/harness/gitness/registry/app/pkg/commons"
)

func (h *Handler) PutArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.GetArtifactInfo(r, true)
	if err != nil {
		handleErrors(ctx, []error{err}, w, nil)
		return
	}
	result := h.Controller.PutArtifact(ctx, info, r.Body)
	if !commons.IsEmpty(result.GetErrors()) {
		handleErrors(ctx, result.GetErrors(), w, result.ResponseHeaders)
		return
	}

	result.ResponseHeaders.WriteToResponse(w)
}

type ReaderFile struct {
	*bytes.Buffer
}
