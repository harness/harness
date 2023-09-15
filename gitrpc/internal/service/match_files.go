// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"
)

func (s RepositoryService) MatchFiles(
	ctx context.Context,
	request *rpc.MatchFilesRequest,
) (*rpc.MatchFilesResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	matchedFiles, err := s.adapter.MatchFiles(ctx, repoPath,
		request.Ref, request.DirPath, request.Pattern, int(request.MaxSize))
	if err != nil {
		return nil, processGitErrorf(err, "failed to open repo")
	}

	files := make([]*rpc.FileContent, len(matchedFiles))
	for i, matchedFile := range matchedFiles {
		files[i] = &rpc.FileContent{
			Path:    matchedFile.Path,
			Content: matchedFile.Content,
		}
	}

	return &rpc.MatchFilesResponse{
		Files: files,
	}, nil
}
