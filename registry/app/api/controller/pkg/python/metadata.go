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

package python

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/python"
	"github.com/harness/gitness/registry/app/pkg/response"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	registrytypes "github.com/harness/gitness/registry/types"
)

// Metadata represents the metadata of a Python package.
func (c *controller) GetPackageMetadata(ctx context.Context, info pythontype.ArtifactInfo) (
	pythontype.PackageMetadata,
	error,
) {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID

		pythonRegistry, ok := a.(python.Registry)
		if !ok {
			return &GetMetadataResponse{
				[]error{fmt.Errorf("invalid registry type: expected python.Registry")},
				nil, pythontype.PackageMetadata{},
			}
		}

		metadata, err := pythonRegistry.GetPackageMetadata(ctx, info)
		return &GetMetadataResponse{
			[]error{err}, nil, metadata,
		}
	}

	result := base.ProxyWrapper(ctx, c.registryDao, f, info.BaseArtifactInfo())
	metadataResponse, ok := result.(*GetMetadataResponse)
	if !ok {
		return pythontype.PackageMetadata{},
			fmt.Errorf("invalid response type: expected GetMetadataResponse, got %T", result)
	}
	return metadataResponse.PackageMetadata, nil
}
