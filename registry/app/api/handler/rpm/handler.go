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

package rpm

import (
	"net/http"

	rpm "github.com/harness/gitness/registry/app/api/controller/pkg/rpm"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
)

type Handler interface {
	pkg.ArtifactInfoProvider
	UploadPackageFile(writer http.ResponseWriter, request *http.Request)
	GetRepoData(writer http.ResponseWriter, request *http.Request)
	DownloadPackageFile(http.ResponseWriter, *http.Request)
}

type handler struct {
	packages.Handler
	controller rpm.Controller
}

func NewHandler(
	controller rpm.Controller,
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

	return &rpmtype.ArtifactInfo{
		ArtifactInfo: info,
	}, nil
}
