// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

func (s RepositoryService) GetBlob(ctx context.Context, request *rpc.GetBlobRequest) (*rpc.GetBlobResponse, error) {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitBlob, err := s.adapter.GetBlob(ctx, repoPath, request.GetSha(), request.GetSizeLimit())
	if err != nil {
		return nil, processGitErrorf(err, "failed to get blob")
	}

	return &rpc.GetBlobResponse{
		Blob: &rpc.Blob{
			Sha:     request.GetSha(),
			Size:    gitBlob.Size,
			Content: gitBlob.Content,
		},
	}, nil
}
