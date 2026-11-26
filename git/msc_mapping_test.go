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
	"testing"
	"time"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

func TestMscMapBranchAndCommit(t *testing.T) {
	t.Parallel()

	shaVal := sha.Must("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	when := time.Unix(1_700_000_000, 0).UTC()

	_, err := mapBranch(nil)
	require.Error(t, err)

	branch, err := mapBranch(&api.Branch{Name: "main", SHA: shaVal})
	require.NoError(t, err)
	require.Equal(t, "main", branch.Name)
	require.Nil(t, branch.Commit)

	_, err = mapCommit(nil)
	require.Error(t, err)

	commit, err := mapCommit(&api.Commit{
		SHA: shaVal, Title: "subject", Message: "body",
		Author:    api.Signature{Identity: api.Identity{Name: "A", Email: "a@test.invalid"}, When: when},
		Committer: api.Signature{Identity: api.Identity{Name: "C", Email: "c@test.invalid"}, When: when},
		FileStats: []api.CommitFileStats{{ChangeType: enum.FileDiffStatusModified, Path: "x", Insertions: 1, Deletions: 2}},
	})
	require.NoError(t, err)
	require.Equal(t, "subject", commit.Title)
	require.Len(t, commit.FileStats, 1)
	require.Equal(t, int64(1), commit.FileStats[0].Insertions)
}

func TestMscMapTreeNodeAndSortOptions(t *testing.T) {
	t.Parallel()

	shaVal := sha.Must("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")

	_, err := mapTreeNode(nil)
	require.Error(t, err)

	node, err := mapTreeNode(&api.TreeNode{
		NodeType: api.TreeNodeTypeBlob,
		Mode:     api.TreeNodeModeFile,
		SHA:      shaVal,
		Name:     "a.txt",
		Path:     "a.txt",
	})
	require.NoError(t, err)
	require.Equal(t, TreeNodeTypeBlob, node.Type)
	require.Equal(t, TreeNodeModeFile, node.Mode)
	require.Equal(t, "a.txt", node.Name)

	_, err = mapTreeNodeType(api.TreeNodeType(99))
	require.Error(t, err)

	_, err = mapTreeNodeMode(api.TreeNodeMode(99))
	require.Error(t, err)

	require.Equal(t, api.GitReferenceFieldObjectName, mapBranchesSortOption(BranchSortOptionName))
	require.Equal(t, api.GitReferenceField(""), mapBranchesSortOption(BranchSortOption(99)))
	require.Equal(t, api.GitReferenceFieldCreatorDate, mapListCommitTagsSortOption(TagSortOptionDate))
	require.Equal(t, api.GitReferenceFieldRefName, mapListCommitTagsSortOption(TagSortOption(99)))

	require.Equal(t, api.SortOrderAsc, mapToSortOrder(SortOrderAsc))
	require.Equal(t, api.SortOrderDefault, mapToSortOrder(SortOrder(99)))
}

func TestMscMapRenameAndDiffHeaders(t *testing.T) {
	t.Parallel()

	details := mapRenameDetails([]api.PathRenameDetails{{
		OldPath: "old", Path: "new",
		CommitSHABefore: sha.Must("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"),
		CommitSHAAfter:  sha.Must("b1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b1"),
	}})
	require.Len(t, details, 1)
	require.Equal(t, "old", details[0].OldPath)

	hunk := mapHunkHeader(&parser.HunkHeader{OldLine: 1, OldSpan: 2, NewLine: 3, NewSpan: 4, Text: " ctx"})
	require.Equal(t, 3, hunk.NewLine)

	fileHeader := mapDiffFileHeader(&parser.DiffFileHeader{OldFileName: "a", NewFileName: "b"})
	require.Equal(t, "a", fileHeader.OldName)
	require.Equal(t, "b", fileHeader.NewName)

	tagSHA := sha.Must("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	tag := mapAnnotatedTag(&api.Tag{Name: "v1", Sha: tagSHA, Title: "release"})
	require.True(t, tag.IsAnnotated)
	require.Equal(t, "v1", tag.Name)
}
