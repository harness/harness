// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"
)

func (s RepositoryService) GetBlob(ctx context.Context, request *rpc.GetBlobRequest) (*rpc.GetBlobResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())
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
