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

package nuget

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/registry/app/api/handler/utils"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

func (h *handler) UploadSymbolPackage(w http.ResponseWriter, r *http.Request) {
	fileReader, _, err := utils.GetFileReader(r, "package")
	if err != nil {
		h.HandleErrors2(r.Context(), errcode.ErrCodeInvalidRequest.WithMessage(fmt.Sprintf("failed to parse file: %s, "+
			"please provide correct file path ", err.Error())), w)
		return
	}
	defer fileReader.Close()
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(*nugettype.ArtifactInfo)

	if !ok {
		log.Ctx(ctx).Error().Msg("Failed to get artifact info from context")
		h.HandleErrors(r.Context(), []error{fmt.Errorf("failed to fetch info from context")}, w)
		return
	}
	response := h.controller.UploadPackage(r.Context(), *info, fileReader, nuget.SymbolsFile)
	if response.GetError() != nil {
		h.HandleError(ctx, w, response.GetError())
		return
	}
	response.ResponseHeaders.WriteToResponse(w)
}
