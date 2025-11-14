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

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/gopackage"
	"github.com/harness/gitness/registry/app/pkg/response"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) RegeneratePackageMetadata(
	ctx context.Context, info *gopackagetype.ArtifactInfo,
) *RegeneratePackageMetadataResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		goRegistry, ok := a.(gopackage.Registry)
		if !ok {
			return c.getRegeneratePackageMetadataErrorResponse(
				fmt.Errorf("invalid registry type: expected go.Registry"),
				nil,
			)
		}
		headers, err := goRegistry.RegeneratePackageMetadata(ctx, *info)
		return c.getRegeneratePackageMetadataErrorResponse(
			err,
			headers,
		)
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, c.quarantineFinder, f, info, false)

	if err != nil {
		return c.getRegeneratePackageMetadataErrorResponse(
			err,
			nil,
		)
	}

	putResponse, ok := result.(*RegeneratePackageMetadataResponse)
	if !ok {
		return c.getRegeneratePackageMetadataErrorResponse(
			fmt.Errorf("invalid response type: expected RegeneratePackageMetadataResponse"),
			nil,
		)
	}
	putResponse.Ok = true
	return putResponse
}

func (c *controller) getRegeneratePackageMetadataErrorResponse(
	err error, headers *commons.ResponseHeaders,
) *RegeneratePackageMetadataResponse {
	return &RegeneratePackageMetadataResponse{
		BaseResponse: BaseResponse{
			err,
			headers,
		},
		Ok: false,
	}
}
