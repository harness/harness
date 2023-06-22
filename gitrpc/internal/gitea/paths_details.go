// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/gitrpc/internal/types"

	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
	gogitfilemode "github.com/go-git/go-git/v5/plumbing/filemode"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

// PathsDetails returns additional details about provided the paths.
func (g Adapter) PathsDetails(ctx context.Context,
	repoPath string,
	rev string,
	paths []string,
) ([]types.PathDetails, error) {
	repoEntry, err := g.repoCache.Get(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	repo := repoEntry.Repo()

	refSHA, err := repo.ResolveRevision(gogitplumbing.Revision(rev))
	if errors.Is(err, gogitplumbing.ErrReferenceNotFound) {
		return nil, types.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to resolve revision %s: %w", rev, err)
	}

	refCommit, err := repo.CommitObject(*refSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to load commit data: %w", err)
	}

	tree, err := refCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for the commit: %w", err)
	}

	results := make([]types.PathDetails, len(paths))

	for i, path := range paths {
		results[i].Path = path

		if len(path) > 0 {
			entry, err := tree.FindEntry(path)
			if errors.Is(err, gogitobject.ErrDirectoryNotFound) || errors.Is(err, gogitobject.ErrEntryNotFound) {
				return nil, types.ErrPathNotFound
			} else if err != nil {
				return nil, fmt.Errorf("can't find path entry %s: %w", path, err)
			}

			if entry.Mode == gogitfilemode.Regular || entry.Mode == gogitfilemode.Executable {
				blobObj, err := repo.Object(gogitplumbing.BlobObject, entry.Hash)
				if err != nil {
					return nil, fmt.Errorf("failed to get blob object size for the path %s and hash %s: %w",
						path, entry.Hash.String(), err)
				}

				results[i].Size = blobObj.(*gogitobject.Blob).Size
			}
		}

		commitEntry, err := g.lastCommitCache.Get(ctx, makeCommitEntryKey(repoPath, refSHA.String(), path))
		if err != nil {
			return nil, fmt.Errorf("failed to find last commit for path %s: %w", path, err)
		}

		results[i].LastCommit = commitEntry
	}

	return results, nil
}
