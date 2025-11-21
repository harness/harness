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

func (c *controller) UpdateYank(
	ctx context.Context, info *cargotype.ArtifactInfo,
	yank bool,
) (*UpdateYankResponse, error) {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		cargoRegistry, ok := a.(cargo.Registry)
		if !ok {
			return &UpdateYankResponse{
				BaseResponse{
					Error:           fmt.Errorf("invalid registry type: expected cargo.Registry"),
					ResponseHeaders: nil,
				}, false,
			}
		}
		headers, err := cargoRegistry.UpdateYank(ctx, *info, yank)
		return &UpdateYankResponse{
			BaseResponse{
				Error:           err,
				ResponseHeaders: headers,
			}, false,
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)

	if err != nil {
		return &UpdateYankResponse{
			BaseResponse{
				Error:           err,
				ResponseHeaders: nil,
			}, false,
		}, err
	}

	putResponse, ok := result.(*UpdateYankResponse)
	if !ok {
		return &UpdateYankResponse{
			BaseResponse: BaseResponse{
				Error:           fmt.Errorf("invalid response type: expected UpdateYankResponse"),
				ResponseHeaders: nil,
			},
			Ok: false,
		}, fmt.Errorf("invalid response type: expected UpdateYankResponse")
	}
	putResponse.Ok = true
	return putResponse, nil
}
