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
	"net/http"
	"strings"

	"github.com/harness/gitness/registry/app/api/controller/pkg/nuget"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"

	"github.com/go-chi/chi/v5"
)

type Handler interface {
	pkg.ArtifactInfoProvider
	UploadPackage(writer http.ResponseWriter, request *http.Request)
	UploadSymbolPackage(writer http.ResponseWriter, request *http.Request)
	DownloadPackage(http.ResponseWriter, *http.Request)
	DeletePackage(writer http.ResponseWriter, request *http.Request)
	GetServiceEndpoint(http.ResponseWriter, *http.Request)
	GetServiceEndpointV2(http.ResponseWriter, *http.Request)
	ListPackageVersion(http.ResponseWriter, *http.Request)
	ListPackageVersionV2(http.ResponseWriter, *http.Request)
	GetPackageMetadata(http.ResponseWriter, *http.Request)
	GetPackageVersionMetadataV2(http.ResponseWriter, *http.Request)
	GetPackageVersionMetadata(http.ResponseWriter, *http.Request)
	SearchPackage(http.ResponseWriter, *http.Request)
	SearchPackageV2(http.ResponseWriter, *http.Request)
	CountPackageV2(http.ResponseWriter, *http.Request)
	GetPackageVersionCountV2(http.ResponseWriter, *http.Request)
	GetServiceMetadataV2(http.ResponseWriter, *http.Request)
}

type handler struct {
	packages.Handler
	controller nuget.Controller
}

func NewHandler(
	controller nuget.Controller,
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

	image := chi.URLParam(r, "id")
	filename := chi.URLParam(r, "filename")
	version := chi.URLParam(r, "version")
	proxyEndpoint := r.URL.Query().Get("proxy_endpoint")
	if image == "" {
		image = r.URL.Query().Get("id")
		image = strings.TrimPrefix(image, "'")
		image = strings.TrimSuffix(image, "'")
	}

	var md nugetmetadata.Metadata

	info.Image = image
	return &nugettype.ArtifactInfo{
		ArtifactInfo:  info,
		Metadata:      md,
		Filename:      filename,
		Version:       version,
		ProxyEndpoint: proxyEndpoint,
		NestedPath:    strings.TrimSuffix(r.PathValue("*"), "/"),
	}, nil
}
