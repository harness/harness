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

package git

import (
	"context"
	"fmt"

	"github.com/harness/gitness/git/api"
)

type MatchFilesParams struct {
	ReadParams
	Ref     string
	DirPath string
	Pattern string
	MaxSize int
}

type MatchFilesOutput struct {
	Files []api.FileContent
}

func (s *Service) MatchFiles(ctx context.Context,
	params *MatchFilesParams,
) (*MatchFilesOutput, error) {
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	matchedFiles, err := s.git.MatchFiles(ctx, repoPath,
		params.Ref, params.DirPath, params.Pattern, params.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("MatchFiles: failed to open repo: %w", err)
	}

	return &MatchFilesOutput{
		Files: matchedFiles,
	}, nil
}
