// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

func (s RepositoryService) GetSubmodule(ctx context.Context,
	request *rpc.GetSubmoduleRequest) (*rpc.GetSubmoduleResponse, error) {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitSubmodule, err := s.adapter.GetSubmodule(ctx, repoPath, request.GetGitRef(), request.GetPath())
	if err != nil {
		return nil, processGitErrorf(err, "failed to get submodule")
	}

	return &rpc.GetSubmoduleResponse{
		Submodule: &rpc.Submodule{
			Name: gitSubmodule.Name,
			Url:  gitSubmodule.URL,
		},
	}, nil
}
