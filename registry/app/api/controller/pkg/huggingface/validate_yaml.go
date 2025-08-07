//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package huggingface

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/huggingface"
	"github.com/harness/gitness/registry/app/pkg/response"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) ValidateYaml(
	ctx context.Context,
	info hftype.ArtifactInfo,
	body io.ReadCloser,
) *ValidateYamlResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		hfRegistry, ok := a.(huggingface.Registry)
		if !ok {
			return &ValidateYamlResponse{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected huggingface.Registry"),
					nil,
				}, nil,
			}
		}
		headers, validateYamlResponse, err := hfRegistry.ValidateYaml(ctx, info, body)
		return &ValidateYamlResponse{
			BaseResponse{
				err,
				headers,
			}, validateYamlResponse,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, f, info)

	if err != nil {
		return &ValidateYamlResponse{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}
	validateYamlResponse, ok := result.(*ValidateYamlResponse)
	if !ok {
		return &ValidateYamlResponse{
			BaseResponse{
				fmt.Errorf("invalid response type: expected ValidateYamlResponse"),
				nil,
			}, nil,
		}
	}
	return validateYamlResponse
}
