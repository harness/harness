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
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/response"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) DownloadPackage(
	ctx context.Context,
	info *cargotype.ArtifactInfo,
) *GetPackageResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		info.Registry = registry
		cargoRegistry, ok := a.(cargo.Registry)
		if !ok {
			return c.getDownloadPackageErrorResponse(
				fmt.Errorf("invalid registry type: expected cargo.Registry"),
				nil,
			)
		}
		headers, fileReader, readCloser, redirectURL, err := cargoRegistry.DownloadPackage(
			ctx, *info,
		)
		return &GetPackageResponse{
			DownloadFileResponse{
				BaseResponse: BaseResponse{
					err,
					headers,
				},
				RedirectURL: redirectURL,
				Body:        fileReader,
				ReadCloser:  readCloser,
			},
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, f, info)
	if err != nil {
		return c.getDownloadPackageErrorResponse(
			err,
			nil,
		)
	}
	getResponse, ok := result.(*GetPackageResponse)
	if !ok {
		return c.getDownloadPackageErrorResponse(
			fmt.Errorf("invalid response type: expected GetPackageResponse"),
			nil,
		)
	}
	return getResponse
}

func (c *controller) getDownloadPackageErrorResponse(
	err error, headers *commons.ResponseHeaders,
) *GetPackageResponse {
	return &GetPackageResponse{
		DownloadFileResponse{
			BaseResponse: BaseResponse{
				err,
				headers},
			RedirectURL: "",
			Body:        nil,
			ReadCloser:  nil,
		},
	}
}
