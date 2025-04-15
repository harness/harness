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

package npm

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	npm2 "github.com/harness/gitness/registry/app/pkg/npm"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/types"
)

// GetPackageMetadata fetches the metadata of a package with the given artifact info.
//
// This is a proxy function to Registry.UploadPackageFile, which is used to fetch the metadata
// of a package. The function is used by the API handler.
//
// The function takes a context and an artifact info as parameters and returns a GetPackageMetadata
// containing the metadata of the package. If an error occurs during the operation, the
// function returns an error as well.
func (c *controller) GetPackageMetadata(
	ctx context.Context,
	info npm.ArtifactInfo,
) *GetMetadataResponse {
	f := func(registry types.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		info.Registry = registry
		info.ParentID = registry.ParentID
		npmRegistry, ok := a.(npm2.Registry)
		if !ok {
			return &GetMetadataResponse{
				BaseResponse: BaseResponse{Error: fmt.Errorf("invalid registry type: expected npm.Registry")},
			}
		}
		metadata, err := npmRegistry.GetPackageMetadata(ctx, info)
		return &GetMetadataResponse{
			BaseResponse:    BaseResponse{Error: err},
			PackageMetadata: metadata,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, f, info)
	if !commons.IsEmpty(err) {
		return &GetMetadataResponse{
			BaseResponse: BaseResponse{Error: err},
		}
	}
	metadataResponse, ok := result.(*GetMetadataResponse)
	if !ok {
		return &GetMetadataResponse{
			BaseResponse: BaseResponse{Error: fmt.Errorf("invalid response type: expected GetMetadataResponse, got %T", result)},
		}
	}
	return metadataResponse
}
