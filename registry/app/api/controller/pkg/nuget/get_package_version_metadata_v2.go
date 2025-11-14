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
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	"github.com/harness/gitness/registry/app/pkg/response"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) GetPackageVersionMetadataV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) *GetPackageVersionMetadataV2Response {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		nugetRegistry, ok := a.(nuget.Registry)
		if !ok {
			return &GetPackageVersionMetadataV2Response{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected nuget.Registry"),
					nil,
				}, nil,
			}
		}
		packageMetadata, err := nugetRegistry.GetPackageVersionMetadataV2(ctx, info)
		return &GetPackageVersionMetadataV2Response{
			BaseResponse{
				err,
				nil,
			}, packageMetadata,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, c.quarantineFinder, f, info, false)

	if err != nil {
		return &GetPackageVersionMetadataV2Response{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}
	packageMetadataResponse, ok := result.(*GetPackageVersionMetadataV2Response)
	if !ok {
		return &GetPackageVersionMetadataV2Response{
			BaseResponse{
				fmt.Errorf("invalid response type: expected GetPackageVersionMetadataV2Response"),
				nil,
			}, nil,
		}
	}
	return packageMetadataResponse
}
