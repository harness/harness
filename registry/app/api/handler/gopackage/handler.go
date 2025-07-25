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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
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
	RegeneratePackageIndex(writer http.ResponseWriter, request *http.Request)
	RegeneratePackageMetadata(writer http.ResponseWriter, request *http.Request)
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

func (h *handler) parseDataFromPayload(r *http.Request) (*bytes.Buffer, *bytes.Buffer, io.ReadCloser, error) {
	var (
		infoBytes = &bytes.Buffer{}
		modBytes  = &bytes.Buffer{}
		zipRC     *multipart.Part
	)

	reader, err := r.MultipartReader()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error reading multipart: %w", err)
	}

	for {
		if infoBytes.Len() > 0 && modBytes.Len() > 0 && zipRC != nil {
			break
		}
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error reading multipart: %w", err)
		}

		switch part.FormName() {
		case "info":
			if _, err := io.Copy(infoBytes, part); err != nil {
				return nil, nil, nil, fmt.Errorf("error reading 'info': %w", err)
			}
			part.Close()
		case "mod":
			if _, err := io.Copy(modBytes, part); err != nil {
				return nil, nil, nil, fmt.Errorf("error reading 'mod': %w", err)
			}
			part.Close()
		case "zip":
			zipRC = part
		default:
			part.Close()
		}
	}

	if infoBytes.Len() == 0 {
		return nil, nil, nil, fmt.Errorf("'info' part not found")
	}

	if modBytes.Len() == 0 {
		return nil, nil, nil, fmt.Errorf("'mod' part not found")
	}

	if zipRC == nil {
		return nil, nil, nil, fmt.Errorf("'zip' part not found")
	}

	return infoBytes, modBytes, zipRC, nil
}
