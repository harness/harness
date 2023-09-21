// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitrpc

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/rpc"
)

type GetSubmoduleParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF string
	Path   string
}

type GetSubmoduleOutput struct {
	Submodule Submodule
}
type Submodule struct {
	Name string
	URL  string
}

func (c *Client) GetSubmodule(ctx context.Context, params *GetSubmoduleParams) (*GetSubmoduleOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	resp, err := c.repoService.GetSubmodule(ctx, &rpc.GetSubmoduleRequest{
		Base:   mapToRPCReadRequest(params.ReadParams),
		GitRef: params.GitREF,
		Path:   params.Path,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to get submodule from server")
	}
	if resp.GetSubmodule() == nil {
		return nil, fmt.Errorf("rpc submodule is nil")
	}

	return &GetSubmoduleOutput{
		Submodule: Submodule{
			Name: resp.GetSubmodule().Name,
			URL:  resp.GetSubmodule().Url,
		},
	}, nil
}
