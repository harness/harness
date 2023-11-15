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
	"errors"
	"fmt"

	"github.com/harness/gitness/git/types"

	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
	gogitfilemode "github.com/go-git/go-git/v5/plumbing/filemode"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

// PathsDetails returns additional details about provided the paths.
//
//nolint:gocognit
func (a Adapter) PathsDetails(ctx context.Context,
	repoPath string,
	ref string,
	paths []string,
) ([]types.PathDetails, error) {
	repo, refCommit, err := a.getGoGitCommit(ctx, repoPath, ref)
	if err != nil {
		return nil, err
	}

	refSHA := refCommit.Hash.String()

	tree, err := refCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for the commit: %w", err)
	}

	results := make([]types.PathDetails, len(paths))

	for i, path := range paths {
		results[i].Path = path

		// use cleaned-up path for calculations to avoid not-founds.
		path = cleanTreePath(path)

		//nolint:nestif
		if len(path) > 0 {
			entry, err := tree.FindEntry(path)
			if errors.Is(err, gogitobject.ErrDirectoryNotFound) || errors.Is(err, gogitobject.ErrEntryNotFound) {
				return nil, &types.PathNotFoundError{Path: path}
			}
			if err != nil {
				return nil, fmt.Errorf("failed to find path entry %s: %w", path, err)
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

		commitEntry, err := a.lastCommitCache.Get(ctx, makeCommitEntryKey(repoPath, refSHA, path))
		if err != nil {
			return nil, fmt.Errorf("failed to find last commit for path %s: %w", path, err)
		}

		results[i].LastCommit = commitEntry
	}

	return results, nil
}
