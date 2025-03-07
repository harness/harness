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

package pypi

import (
	"net/http"

	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/utils"
	pypi2 "github.com/harness/gitness/registry/app/metadata/pypi"
	"github.com/harness/gitness/registry/app/pkg/pypi"

	"github.com/go-chi/chi/v5"
)

type Handler interface {
	DownloadPackageFile(http.ResponseWriter, *http.Request)
	UploadPackageFile(writer http.ResponseWriter, request *http.Request)
	PackageMetadata(writer http.ResponseWriter, request *http.Request)
}

type handler struct {
	packages.Handler
	controller pypi.Controller
}

func NewHandler(
	controller pypi.Controller,
	packageHandler packages.Handler,
) Handler {
	return &handler{
		Handler:    packageHandler,
		controller: controller,
	}
}

var _ Handler = (*handler)(nil)

func (h *handler) getPackageArtifactInfo(r *http.Request) (pypi.ArtifactInfo, error) {
	info, e := h.GetArtifactInfo(r)

	if e.Error() != "" {
		return pypi.ArtifactInfo{}, e
	}

	var md pypi2.Metadata
	err := utils.FillFromForm(r, &md)
	if err != nil {
		return pypi.ArtifactInfo{}, err
	}

	info.Image = chi.URLParam(r, "image")
	if info.Image == "" {
		info.Image = md.Name
	}

	return pypi.ArtifactInfo{
		ArtifactInfo: &info,
		Metadata:     md,
	}, nil
}
