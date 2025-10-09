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
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/gopackage"
	"github.com/harness/gitness/registry/app/pkg/response"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) UploadPackage(
	ctx context.Context, info *gopackagetype.ArtifactInfo,
	mod io.ReadCloser, zip io.ReadCloser,
) *UploadFileResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		gopackageRegistry, ok := a.(gopackage.Registry)
		if !ok {
			return c.getUploadPackageFileErrorResponse(
				fmt.Errorf("invalid registry type: expected gopackage.Registry"),
				nil,
			)
		}
		headers, err := gopackageRegistry.UploadPackage(ctx, *info, mod, zip)
		return c.getUploadPackageFileErrorResponse(
			err,
			headers,
		)
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)

	if err != nil {
		return c.getUploadPackageFileErrorResponse(
			err,
			nil,
		)
	}

	putResponse, ok := result.(*UploadFileResponse)
	if !ok {
		return c.getUploadPackageFileErrorResponse(
			fmt.Errorf("invalid response type: expected BaseResponse"),
			nil,
		)
	}
	putResponse.ResponseHeaders.Code = http.StatusOK
	putResponse.Status = "success"
	putResponse.Image = info.Image
	putResponse.Version = info.Version
	return putResponse
}

func (c *controller) getUploadPackageFileErrorResponse(
	err error, headers *commons.ResponseHeaders,
) *UploadFileResponse {
	return &UploadFileResponse{
		BaseResponse: BaseResponse{
			err,
			headers,
		},
	}
}
