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

package gopackage

import (
	"net/http"

	"github.com/harness/gitness/registry/app/api/controller/pkg/gopackage"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
)

type Handler interface {
	pkg.ArtifactInfoProvider
	UploadPackage(writer http.ResponseWriter, request *http.Request)
	DownloadPackageFile(writer http.ResponseWriter, request *http.Request)
}

type handler struct {
	packages.Handler
	controller gopackage.Controller
}

func NewHandler(
	controller gopackage.Controller,
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
	return &gopackagetype.ArtifactInfo{
		ArtifactInfo: info,
		Version:      r.PathValue("version"),
	}, nil
}

func (h *handler) handleGoPackageAPIError(writer http.ResponseWriter, request *http.Request, err error) {
	h.HandleErrors(request.Context(), []error{err}, writer)
}
