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
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/pkg/rpm"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	registrytypes "github.com/harness/gitness/registry/types"
)

// GetRepoData represents the metadata of a RPM package.
func (c *controller) GetRepoData(ctx context.Context, info rpmtype.ArtifactInfo, fileName string) *GetRepoDataResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		info.Registry = registry
		rpmRegistry, ok := a.(rpm.Registry)
		if !ok {
			return &GetRepoDataResponse{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected rpm.Registry"),
					nil,
				},
				"", nil, nil,
			}
		}

		responseHeaders, fileReader, _, redirectURL, err := rpmRegistry.GetRepoData(ctx, info, fileName)

		return &GetRepoDataResponse{
			BaseResponse{
				err,
				responseHeaders,
			},
			redirectURL, fileReader, fileReader,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, f, info)
	metadataResponse, ok := result.(*GetRepoDataResponse)
	if !ok {
		return &GetRepoDataResponse{
			BaseResponse{
				err,
				nil,
			},
			"", nil, nil,
		}
	}
	return metadataResponse
}
