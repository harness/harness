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

// CommitEntry represents a single line in the commits input.
// Make sure these structs are defined within this package or imported correctly.
// If they are already defined in types.go or similar in this package, remove these definitions.

func (c *controller) CommitRevision(
	ctx context.Context,
	info hftype.ArtifactInfo,
	body io.ReadCloser) *CommitRevisionResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		hfRegistry, ok := a.(huggingface.Registry)
		if !ok {
			return &CommitRevisionResponse{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected huggingface.Registry"),
					nil,
				}, nil,
			}
		}
		headers, commitRevisionResponse, err := hfRegistry.CommitRevision(ctx, info, body)
		return &CommitRevisionResponse{
			BaseResponse{
				err,
				headers,
			}, commitRevisionResponse,
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.registryDao, f, info)

	if err != nil {
		return &CommitRevisionResponse{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}
	commitRevisionResponse, ok := result.(*CommitRevisionResponse)
	if !ok {
		return &CommitRevisionResponse{
			BaseResponse{
				fmt.Errorf("invalid response type: expected CommitRevisionResponse"),
				nil,
			}, nil,
		}
	}
	return commitRevisionResponse
}
