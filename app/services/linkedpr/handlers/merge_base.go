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

package handlers

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

// RunMergeBase calls git.MergeBase against the linked repo's local mirror.
// Both refs must already exist locally.
func RunMergeBase(
	ctx context.Context,
	gitClient git.Interface,
	repoFinder refcache.RepoFinder,
	linkedRepo *types.LinkedRepo,
	ref1, ref2 string,
) (string, error) {
	repo, err := repoFinder.FindByID(ctx, linkedRepo.RepoID)
	if err != nil {
		return "", fmt.Errorf("linkedpr: find repo %d: %w", linkedRepo.RepoID, err)
	}
	out, err := gitClient.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: git.ReadParams{RepoUID: repo.GitUID},
		Ref1:       ref1,
		Ref2:       ref2,
	})
	if err != nil {
		return "", fmt.Errorf("linkedpr: git MergeBase: %w", err)
	}
	return out.MergeBaseSHA.String(), nil
}
