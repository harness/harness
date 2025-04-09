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
	"github.com/harness/gitness/registry/types"
)

func (c *controller) ListTags(
	ctx context.Context,
	info npm.ArtifactInfo,
) *ListTagResponse {
	f := func(registry types.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		npmRegistry, ok := a.(npm2.Registry)
		if !ok {
			return &ListTagResponse{
				BaseResponse: BaseResponse{Error: fmt.Errorf("invalid registry type: expected npm.Registry")},
			}
		}
		tags, err := npmRegistry.ListTags(ctx, info)
		if !commons2.IsEmpty(err) {
			return &ListTagResponse{
				BaseResponse: BaseResponse{Error: err},
			}
		}
		return &ListTagResponse{
			Tags: tags,
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)
	if !commons2.IsEmpty(err) {
		return &ListTagResponse{
			BaseResponse: BaseResponse{Error: err},
		}
	}
	artifactResponse, ok := result.(*ListTagResponse)
	if !ok {
		return &ListTagResponse{
			BaseResponse: BaseResponse{Error: fmt.Errorf("invalid response type: expected ListTagResponse")},
		}
	}
	return artifactResponse
}
