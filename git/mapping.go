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
	"fmt"

	"github.com/harness/gitness/git/types"
)

func mapBranch(b *types.Branch) (*Branch, error) {
	if b == nil {
		return nil, fmt.Errorf("rpc branch is nil")
	}

	var commit *Commit
	if b.Commit != nil {
		var err error
		commit, err = mapCommit(b.Commit)
		if err != nil {
			return nil, err
		}
	}

	return &Branch{
		Name:   b.Name,
		SHA:    b.SHA,
		Commit: commit,
	}, nil
}

func mapCommit(c *types.Commit) (*Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("rpc commit is nil")
	}

	author, err := mapSignature(&c.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc author: %w", err)
	}

	comitter, err := mapSignature(&c.Committer)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc committer: %w", err)
	}
	return &Commit{
		SHA:        c.SHA,
		ParentSHAs: c.ParentSHAs,
		Title:      c.Title,
		Message:    c.Message,
		Author:     *author,
		Committer:  *comitter,
		FileStats:  *mapFileStats(&c.FileStats),
	}, nil
}

func mapFileStats(s *types.CommitFileStats) *CommitFileStats {
	return &CommitFileStats{
		Added:    s.Added,
		Modified: s.Modified,
		Removed:  s.Removed,
	}
}

func mapSignature(s *types.Signature) (*Signature, error) {
	if s == nil {
		return nil, fmt.Errorf("rpc signature is nil")
	}

	identity, err := mapIdentity(&s.Identity)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc identity: %w", err)
	}

	return &Signature{
		Identity: identity,
		When:     s.When,
	}, nil
}

func mapIdentity(id *types.Identity) (Identity, error) {
	if id == nil {
		return Identity{}, fmt.Errorf("rpc identity is nil")
	}

	return Identity{
		Name:  id.Name,
		Email: id.Email,
	}, nil
}

func mapBranchesSortOption(o BranchSortOption) types.GitReferenceField {
	switch o {
	case BranchSortOptionName:
		return types.GitReferenceFieldObjectName
	case BranchSortOptionDate:
		return types.GitReferenceFieldCreatorDate
	case BranchSortOptionDefault:
		fallthrough
	default:
		// no need to error out - just use default for sorting
		return ""
	}
}

func mapAnnotatedTag(tag *types.Tag) *CommitTag {
	tagger, _ := mapSignature(&tag.Tagger)
	return &CommitTag{
		Name:        tag.Name,
		SHA:         tag.Sha,
		Title:       tag.Title,
		Message:     tag.Message,
		Tagger:      tagger,
		IsAnnotated: true,
		Commit:      nil,
	}
}

func mapListCommitTagsSortOption(s TagSortOption) types.GitReferenceField {
	switch s {
	case TagSortOptionDate:
		return types.GitReferenceFieldCreatorDate
	case TagSortOptionName:
		return types.GitReferenceFieldRefName
	case TagSortOptionDefault:
		return types.GitReferenceFieldRefName
	default:
		// no need to error out - just use default for sorting
		return types.GitReferenceFieldRefName
	}
}

func mapTreeNode(n *types.TreeNode) (TreeNode, error) {
	if n == nil {
		return TreeNode{}, fmt.Errorf("rpc tree node is nil")
	}

	nodeType, err := mapTreeNodeType(n.NodeType)
	if err != nil {
		return TreeNode{}, err
	}

	mode, err := mapTreeNodeMode(n.Mode)
	if err != nil {
		return TreeNode{}, err
	}

	return TreeNode{
		Type: nodeType,
		Mode: mode,
		SHA:  n.Sha,
		Name: n.Name,
		Path: n.Path,
	}, nil
}

func mapTreeNodeType(t types.TreeNodeType) (TreeNodeType, error) {
	switch t {
	case types.TreeNodeTypeBlob:
		return TreeNodeTypeBlob, nil
	case types.TreeNodeTypeCommit:
		return TreeNodeTypeCommit, nil
	case types.TreeNodeTypeTree:
		return TreeNodeTypeTree, nil
	default:
		return TreeNodeTypeBlob, fmt.Errorf("unknown rpc tree node type: %d", t)
	}
}

func mapTreeNodeMode(m types.TreeNodeMode) (TreeNodeMode, error) {
	switch m {
	case types.TreeNodeModeFile:
		return TreeNodeModeFile, nil
	case types.TreeNodeModeExec:
		return TreeNodeModeExec, nil
	case types.TreeNodeModeSymlink:
		return TreeNodeModeSymlink, nil
	case types.TreeNodeModeCommit:
		return TreeNodeModeCommit, nil
	case types.TreeNodeModeTree:
		return TreeNodeModeTree, nil
	default:
		return TreeNodeModeFile, fmt.Errorf("unknown rpc tree node mode: %d", m)
	}
}

func mapRenameDetails(c []types.PathRenameDetails) []*RenameDetails {
	renameDetailsList := make([]*RenameDetails, len(c))
	for i, detail := range c {
		renameDetailsList[i] = &RenameDetails{
			OldPath:         detail.OldPath,
			NewPath:         detail.NewPath,
			CommitShaBefore: detail.CommitSHABefore,
			CommitShaAfter:  detail.CommitSHAAfter,
		}
	}
	return renameDetailsList
}

func mapToSortOrder(o SortOrder) types.SortOrder {
	switch o {
	case SortOrderAsc:
		return types.SortOrderAsc
	case SortOrderDesc:
		return types.SortOrderDesc
	case SortOrderDefault:
		return types.SortOrderDefault
	default:
		// no need to error out - just use default for sorting
		return types.SortOrderDefault
	}
}

func mapHunkHeader(h *types.HunkHeader) HunkHeader {
	return HunkHeader{
		OldLine: h.OldLine,
		OldSpan: h.OldSpan,
		NewLine: h.NewLine,
		NewSpan: h.NewSpan,
		Text:    h.Text,
	}
}

func mapDiffFileHeader(h *types.DiffFileHeader) DiffFileHeader {
	return DiffFileHeader{
		OldName:    h.OldFileName,
		NewName:    h.NewFileName,
		Extensions: h.Extensions,
	}
}
