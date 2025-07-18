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

func (c *controller) SearchPackageV2(ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string, limit, offset int) *SearchPackageV2Response {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		nugetRegistry, ok := a.(nuget.Registry)
		if !ok {
			return &SearchPackageV2Response{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected nuget.Registry"),
					nil,
				}, nil,
			}
		}

		feedResponse, err := nugetRegistry.SearchPackageV2(ctx, info, searchTerm, limit, offset)
		return &SearchPackageV2Response{
			BaseResponse{
				err,
				nil,
			}, feedResponse,
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)

	if err != nil {
		return &SearchPackageV2Response{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}
	searchPackageResponse, ok := result.(*SearchPackageV2Response)
	if !ok {
		return &SearchPackageV2Response{
			BaseResponse{
				fmt.Errorf("invalid response type: expected SearchPackageV2Response"),
				nil,
			}, nil,
		}
	}
	return searchPackageResponse
}
