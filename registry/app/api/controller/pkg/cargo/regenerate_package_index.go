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
	"github.com/harness/gitness/registry/app/pkg/response"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) RegeneratePackageIndex(
	ctx context.Context, info *cargotype.ArtifactInfo,
) (*RegeneratePackageIndexResponse, error) {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		cargoRegistry, ok := a.(cargo.Registry)
		if !ok {
			return &RegeneratePackageIndexResponse{
				BaseResponse{
					Error:           fmt.Errorf("invalid registry type: expected cargo.Registry"),
					ResponseHeaders: nil,
				}, false,
			}
		}
		headers, err := cargoRegistry.RegeneratePackageIndex(ctx, *info)
		return &RegeneratePackageIndexResponse{
			BaseResponse{
				Error:           err,
				ResponseHeaders: headers,
			}, false,
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)

	if err != nil {
		return &RegeneratePackageIndexResponse{
			BaseResponse{
				Error:           err,
				ResponseHeaders: nil,
			}, false,
		}, err
	}

	putResponse, ok := result.(*RegeneratePackageIndexResponse)
	if !ok {
		return &RegeneratePackageIndexResponse{
			BaseResponse: BaseResponse{
				Error:           fmt.Errorf("invalid response type: expected RegeneratePackageIndexResponse"),
				ResponseHeaders: nil,
			},
			Ok: false,
		}, fmt.Errorf("invalid response type: expected RegeneratePackageIndexResponse")
	}
	putResponse.Ok = true
	return putResponse, nil
}
