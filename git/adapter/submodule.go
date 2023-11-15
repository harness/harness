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

package adapter

import (
	"context"

	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
)

// GetSubmodule returns the submodule at the given path reachable from ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (a Adapter) GetSubmodule(
	ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) (*types.Submodule, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting commit for ref '%s'", ref)
	}

	giteaSubmodule, err := giteaCommit.GetSubModule(treePath)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting submodule '%s' from commit", treePath)
	}

	return &types.Submodule{
		Name: giteaSubmodule.Name,
		URL:  giteaSubmodule.URL,
	}, nil
}
