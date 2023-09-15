// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/gitrpc/internal/types"

	gogit "github.com/go-git/go-git/v5"
	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

func (g Adapter) getGoGitCommit(ctx context.Context,
	repoPath string,
	rev string,
) (*gogit.Repository, *gogitobject.Commit, error) {
	repoEntry, err := g.repoCache.Get(ctx, repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open repository: %w", err)
	}

	repo := repoEntry.Repo()

	var refSHA *gogitplumbing.Hash
	if rev == "" {
		var head *gogitplumbing.Reference
		head, err = repo.Head()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get head: %w", err)
		}

		headHash := head.Hash()
		refSHA = &headHash
	} else {
		refSHA, err = repo.ResolveRevision(gogitplumbing.Revision(rev))
		if errors.Is(err, gogitplumbing.ErrReferenceNotFound) {
			return nil, nil, types.ErrNotFound
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve revision %s: %w", rev, err)
		}
	}

	refCommit, err := repo.CommitObject(*refSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load commit data: %w", err)
	}

	return repo, refCommit, nil
}
