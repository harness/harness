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
	"net/http"

	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven"

	"github.com/rs/zerolog/log"
)

func (h *Handler) HeadArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.GetArtifactInfo(r, true)
	if err != nil {
		handleErrors(ctx, []error{err}, w)
		return
	}
	result := h.Controller.HeadArtifact(
		ctx,
		info,
	)
	if !commons.IsEmpty(result.GetErrors()) {
		handleErrors(ctx, result.GetErrors(), w)
		return
	}
	response, ok := result.(*maven.HeadArtifactResponse)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to cast result to HeadArtifactResponse")
		return
	}
	response.ResponseHeaders.WriteToResponse(w)
}
