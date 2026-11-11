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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListBranchesIsLastPage(t *testing.T) {
	requireGit(t)

	s, root := newTestService(t)
	ctx := context.Background()

	const uid = "lastpage"
	repoPath := getFullPathForRepo(root, uid)
	require.NoError(t, os.MkdirAll(repoPath, 0o755))
	runGit(t, repoPath, "init", "--bare", "-q")

	tree := runGit(t, repoPath, "hash-object", "-t", "tree", "-w", "/dev/null")
	commit := runGit(t, repoPath, "-c", "user.name=t", "-c", "user.email=t@t", "commit-tree", tree, "-m", "x")
	const total = 6
	for i := 1; i <= total; i++ {
		runGit(t, repoPath, "update-ref", fmt.Sprintf("refs/heads/b%d", i), commit)
	}

	tests := []struct {
		name     string
		page     int32
		pageSize int32
		wantLen  int
		wantLast bool
	}{
		{"partial first page", 1, 4, 4, false},
		{"partial last page", 2, 4, 2, true},
		{"exact multiple, first page", 1, 3, 3, false},
		{"exact multiple, last page", 2, 3, 3, true},
		{"single page holds everything", 1, 6, 6, true},
		{"page past the end", 3, 3, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := s.ListBranches(ctx, &ListBranchesParams{
				ReadParams: ReadParams{RepoUID: uid},
				Page:       tt.page,
				PageSize:   tt.pageSize,
			})
			require.NoError(t, err)
			require.Len(t, out.Branches, tt.wantLen)
			require.Equal(t, tt.wantLast, out.IsLastPage)
		})
	}
}
