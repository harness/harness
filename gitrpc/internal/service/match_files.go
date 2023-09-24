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
