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
	"strings"

	"github.com/harness/gitness/git/merge"

	"golang.org/x/sync/errgroup"
)

type SummaryParams struct {
	ReadParams
}

type SummaryOutput struct {
	CommitCount int
	BranchCount int
	TagCount    int
}

func (s *Service) Summary(
	ctx context.Context,
	params SummaryParams,
) (SummaryOutput, error) {
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	defaultBranch, err := s.git.GetDefaultBranch(ctx, repoPath)
	if err != nil {
		return SummaryOutput{}, err
	}
	defaultBranch = strings.TrimSpace(defaultBranch)

	g, ctx := errgroup.WithContext(ctx)

	var commitCount, branchCount, tagCount int

	g.Go(func() error {
		var err error
		commitCount, err = merge.CommitCount(ctx, repoPath, "", defaultBranch)
		return err
	})

	g.Go(func() error {
		var err error
		branchCount, err = s.git.GetBranchCount(ctx, repoPath)
		return err
	})

	g.Go(func() error {
		var err error
		tagCount, err = s.git.GetTagCount(ctx, repoPath)
		return err
	})

	if err := g.Wait(); err != nil {
		return SummaryOutput{}, fmt.Errorf("failed to get repo summary: %w", err)
	}

	return SummaryOutput{
		CommitCount: commitCount,
		BranchCount: branchCount,
		TagCount:    tagCount,
	}, nil
}
