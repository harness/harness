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

package python

import (
	"net/http"

	"github.com/harness/gitness/registry/app/api/controller/pkg/python"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/utils"
	python2 "github.com/harness/gitness/registry/app/metadata/python"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"

	"github.com/go-chi/chi/v5"
)

type Handler interface {
	pkg.ArtifactInfoProvider
	UploadPackageFile(writer http.ResponseWriter, request *http.Request)
	DownloadPackageFile(http.ResponseWriter, *http.Request)
	PackageMetadata(writer http.ResponseWriter, request *http.Request)
}

type handler struct {
	packages.Handler
	controller python.Controller
}

func NewHandler(
	controller python.Controller,
	packageHandler packages.Handler,
) Handler {
	return &handler{
		Handler:    packageHandler,
		controller: controller,
	}
}

var _ Handler = (*handler)(nil)

func (h *handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	info, err := h.Handler.GetArtifactInfo(r)
	if !commons.IsEmptyError(err) {
		return nil, err
	}

	image := chi.URLParam(r, "image")
	filename := chi.URLParam(r, "filename")
	version := chi.URLParam(r, "version")

	var md python2.Metadata
	err2 := utils.FillFromForm(r, &md)

	if err2 == nil {
		if image == "" {
			image = md.Name
		}
		if version == "" {
			version = md.Version
		}
	}

	info.Image = image
	return &pythontype.ArtifactInfo{
		ArtifactInfo: info,
		Metadata:     md,
		Filename:     filename,
		Version:      version,
	}, nil
}
