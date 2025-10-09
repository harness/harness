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

package packages

import (
	"fmt"
	"net/http"
	"strings"

	commons2 "github.com/harness/gitness/registry/app/pkg/types/commons"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(commons2.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get common artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch common artifact info from context")}, w)
		return
	}
	path := r.FormValue("path")
	if path == "" {
		h.HandleErrors(r.Context(), []error{fmt.Errorf("path parameter is required")}, w)
		return
	}

	registryURL := h.URLProvider.PackageURL(ctx, strings.Join([]string{info.RootIdentifier, info.RegIdentifier}, "/"),
		"", "files", path)

	http.Redirect(w, r, registryURL, http.StatusPermanentRedirect)
}
