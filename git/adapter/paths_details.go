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
	"fmt"

	"github.com/harness/gitness/git/types"
)

// PathsDetails returns additional details about provided the paths.
func (a Adapter) PathsDetails(ctx context.Context,
	repoPath string,
	rev string,
	paths []string,
) ([]types.PathDetails, error) {
	// resolve the git revision to the commit SHA - we need the commit SHA for the last commit hash entry key.
	commitSHA, err := a.ResolveRev(ctx, repoPath, rev)
	if err != nil {
		return nil, fmt.Errorf("failed to get path details: %w", err)
	}

	results := make([]types.PathDetails, len(paths))

	for i, path := range paths {
		results[i].Path = path

		path = cleanTreePath(path) // use cleaned-up path for calculations to avoid not-founds.

		commitEntry, err := a.lastCommitCache.Get(ctx, makeCommitEntryKey(repoPath, commitSHA, path))
		if err != nil {
			return nil, fmt.Errorf("failed to find last commit for path %s: %w", path, err)
		}

		results[i].LastCommit = commitEntry
	}

	return results, nil
}
