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
	"net/http"

	cargo "github.com/harness/gitness/registry/app/api/controller/pkg/cargo"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
)

type Handler interface {
	pkg.ArtifactInfoProvider
	GetRegistryConfig(writer http.ResponseWriter, request *http.Request)
	DownloadPackageIndex(writer http.ResponseWriter, request *http.Request)
	RegeneratePackageIndex(writer http.ResponseWriter, request *http.Request)
	DownloadPackage(writer http.ResponseWriter, request *http.Request)
	SearchPackage(writer http.ResponseWriter, request *http.Request)
	UploadPackage(writer http.ResponseWriter, request *http.Request)
	YankVersion(writer http.ResponseWriter, request *http.Request)
	UnYankVersion(writer http.ResponseWriter, request *http.Request)
}

type handler struct {
	packages.Handler
	controller cargo.Controller
}

func NewHandler(
	controller cargo.Controller,
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
	if err != nil {
		return nil, err
	}
	info.Image = r.PathValue("name")
	return &cargotype.ArtifactInfo{
		ArtifactInfo: info,
		Version:      r.PathValue("version"),
	}, nil
}

func (h *handler) handleCargoPackageAPIError(writer http.ResponseWriter, request *http.Request, err error) {
	h.HandleErrors(request.Context(), []error{err}, writer)
}
