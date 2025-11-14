// Copyright 2023 Harness, Inc.
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

package huggingface

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/huggingface"
	"github.com/harness/gitness/registry/app/pkg/response"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) RevisionInfo(
	ctx context.Context,
	info hftype.ArtifactInfo,
	queryParams map[string][]string,
) *RevisionInfoResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		hfRegistry, ok := a.(huggingface.Registry)
		if !ok {
			return &RevisionInfoResponse{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected huggingface.Registry"),
					nil,
				}, nil,
			}
		}
		headers, revisionInfoResponse, err := hfRegistry.RevisionInfo(ctx, info, queryParams)
		return &RevisionInfoResponse{
			BaseResponse{
				err,
				headers,
			}, revisionInfoResponse,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.registryDao, c.quarantineFinder, f, info, false)

	if err != nil {
		return &RevisionInfoResponse{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}
	revisionInfoResponse, ok := result.(*RevisionInfoResponse)
	if !ok {
		return &RevisionInfoResponse{
			BaseResponse{
				fmt.Errorf("invalid response type: expected RevisionInfoResponse"),
				nil,
			}, nil,
		}
	}
	return revisionInfoResponse
}
