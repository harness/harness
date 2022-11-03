// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/types"
)

// GetSubmodule returns the submodule at the given path reachable from ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) GetSubmodule(ctx context.Context, repoPath string,
	ref string, treePath string) (*types.Submodule, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, fmt.Errorf("error getting commit for ref '%s': %w", ref, err)
	}

	giteaSubmodule, err := giteaCommit.GetSubModule(treePath)
	if err != nil {
		return nil, fmt.Errorf("error getting submodule '%s' from commit: %w", ref, err)
	}

	return &types.Submodule{
		Name: giteaSubmodule.Name,
		URL:  giteaSubmodule.URL,
	}, nil
}
