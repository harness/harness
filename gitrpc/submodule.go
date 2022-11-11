// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/rpc"
)

type GetSubmoduleParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
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
		RepoUid: params.RepoUID,
		GitRef:  params.GitREF,
		Path:    params.Path,
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
