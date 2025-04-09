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
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	npm2 "github.com/harness/gitness/registry/app/pkg/npm"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/types"
)

// UploadPackageFile FIXME: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (c *controller) UploadPackageFile(
	ctx context.Context,
	info npm.ArtifactInfo,
	file io.ReadCloser,
) *PutArtifactResponse {
	f := func(registry types.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		npmRegistry, ok := a.(npm2.Registry)
		if !ok {
			return &PutArtifactResponse{
				BaseResponse: BaseResponse{Error: fmt.Errorf("invalid registry type: expected npm.Registry")},
			}
		}
		headers, sha256, err := npmRegistry.UploadPackageFile(ctx, info, file)
		if !commons.IsEmpty(err) {
			return &PutArtifactResponse{
				BaseResponse: BaseResponse{Error: err},
			}
		}
		return &PutArtifactResponse{
			BaseResponse: BaseResponse{ResponseHeaders: headers},
			Sha256:       sha256}
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)
	if !commons.IsEmpty(err) {
		return &PutArtifactResponse{
			BaseResponse: BaseResponse{Error: err},
		}
	}
	artifactResponse, ok := result.(*PutArtifactResponse)
	if !ok {
		return &PutArtifactResponse{
			BaseResponse: BaseResponse{Error: fmt.Errorf("invalid response type: expected PutArtifactResponse")},
		}
	}
	return artifactResponse
}
