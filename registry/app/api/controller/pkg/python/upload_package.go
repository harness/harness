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
	"mime/multipart"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/python"
	"github.com/harness/gitness/registry/app/pkg/response"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	registrytypes "github.com/harness/gitness/registry/types"
)

// UploadPackageFile FIXME: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (c *controller) UploadPackageFile(
	ctx context.Context,
	info pythontype.ArtifactInfo,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) *PutArtifactResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		pythonRegistry, ok := a.(python.Registry)
		if !ok {
			return &PutArtifactResponse{
				"",
				[]error{fmt.Errorf("invalid registry type: expected python.Registry")},
				nil,
			}
		}
		headers, sha256, err := pythonRegistry.UploadPackageFile(ctx, info, file, fileHeader.Filename)
		if commons.IsEmptyError(err) {
			return &PutArtifactResponse{
				sha256, []error{}, headers,
			}
		}
		return &PutArtifactResponse{
			sha256, []error{err}, headers,
		}
	}

	result := base.NoProxyWrapper(ctx, c.registryDao, f, info.BaseArtifactInfo())
	response, ok := result.(*PutArtifactResponse)
	if !ok {
		return &PutArtifactResponse{
			"",
			[]error{fmt.Errorf("invalid response type: expected PutArtifactResponse")},
			nil,
		}
	}
	return response
}
