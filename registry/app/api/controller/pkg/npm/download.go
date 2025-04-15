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
	commons2 "github.com/harness/gitness/registry/app/pkg/commons"
	npm2 "github.com/harness/gitness/registry/app/pkg/npm"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) DownloadPackageFile(
	ctx context.Context,
	info npm.ArtifactInfo,
) *GetArtifactResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		info.Registry = registry
		info.ParentID = registry.ParentID
		npmRegistry, ok := a.(npm2.Registry)
		if !ok {
			return &GetArtifactResponse{
				BaseResponse: BaseResponse{Error: fmt.Errorf("invalid registry type: expected npm.Registry")},
			}
		}
		headers, fileReader, readCloser, redirectURL, err := npmRegistry.DownloadPackageFile(ctx, info)
		return &GetArtifactResponse{
			BaseResponse: BaseResponse{Error: err, ResponseHeaders: headers},
			RedirectURL:  redirectURL,
			Body:         fileReader,
			ReadCloser:   readCloser,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, f, info)
	if !commons2.IsEmpty(err) {
		return &GetArtifactResponse{
			BaseResponse: BaseResponse{Error: err},
		}
	}
	getResponse, ok := result.(*GetArtifactResponse)
	if !ok {
		return &GetArtifactResponse{
			BaseResponse: BaseResponse{Error: fmt.Errorf("invalid response type: expected GetArtifactResponse")},
		}
	}
	return getResponse
}
