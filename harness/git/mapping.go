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

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/parser"
)

func mapBranch(b *api.Branch) (*Branch, error) {
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

func mapCommit(c *api.Commit) (*Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("rpc commit is nil")
	}

	return &Commit{
		SHA:        c.SHA,
		TreeSHA:    c.TreeSHA,
		ParentSHAs: c.ParentSHAs,
		Title:      c.Title,
		Message:    c.Message,
		Author:     mapSignature(c.Author),
		Committer:  mapSignature(c.Committer),
		SignedData: (*SignedData)(c.SignedData),
		FileStats:  mapFileStats(c.FileStats),
	}, nil
}

func mapFileStats(typeStats []api.CommitFileStats) []CommitFileStats {
	var stats = make([]CommitFileStats, len(typeStats))

	for i, tStat := range typeStats {
		stats[i] = CommitFileStats{
			Status:     tStat.ChangeType,
			Path:       tStat.Path,
			OldPath:    tStat.OldPath,
			Insertions: tStat.Insertions,
			Deletions:  tStat.Deletions,
		}
	}

	return stats
}

func mapSignature(s api.Signature) Signature {
	return Signature{
		Identity: Identity(s.Identity),
		When:     s.When,
	}
}

func mapBranchesSortOption(o BranchSortOption) api.GitReferenceField {
	switch o {
	case BranchSortOptionName:
		return api.GitReferenceFieldObjectName
	case BranchSortOptionDate:
		return api.GitReferenceFieldCreatorDate
	case BranchSortOptionDefault:
		fallthrough
	default:
		// no need to error out - just use default for sorting
		return ""
	}
}

func mapAnnotatedTag(tag *api.Tag) *CommitTag {
	tagger := mapSignature(tag.Tagger)
	return &CommitTag{
		Name:        tag.Name,
		SHA:         tag.Sha,
		Title:       tag.Title,
		Message:     tag.Message,
		Tagger:      &tagger,
		IsAnnotated: true,
		SignedData:  (*SignedData)(tag.SignedData),
		Commit:      nil,
	}
}

func mapListCommitTagsSortOption(s TagSortOption) api.GitReferenceField {
	switch s {
	case TagSortOptionDate:
		return api.GitReferenceFieldCreatorDate
	case TagSortOptionName:
		return api.GitReferenceFieldRefName
	case TagSortOptionDefault:
		return api.GitReferenceFieldRefName
	default:
		// no need to error out - just use default for sorting
		return api.GitReferenceFieldRefName
	}
}

func mapTreeNode(n *api.TreeNode) (TreeNode, error) {
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
		SHA:  n.SHA.String(),
		Name: n.Name,
		Path: n.Path,
	}, nil
}

func mapTreeNodeType(t api.TreeNodeType) (TreeNodeType, error) {
	switch t {
	case api.TreeNodeTypeBlob:
		return TreeNodeTypeBlob, nil
	case api.TreeNodeTypeCommit:
		return TreeNodeTypeCommit, nil
	case api.TreeNodeTypeTree:
		return TreeNodeTypeTree, nil
	default:
		return TreeNodeTypeBlob, fmt.Errorf("unknown rpc tree node type: %d", t)
	}
}

func mapTreeNodeMode(m api.TreeNodeMode) (TreeNodeMode, error) {
	switch m {
	case api.TreeNodeModeFile:
		return TreeNodeModeFile, nil
	case api.TreeNodeModeExec:
		return TreeNodeModeExec, nil
	case api.TreeNodeModeSymlink:
		return TreeNodeModeSymlink, nil
	case api.TreeNodeModeCommit:
		return TreeNodeModeCommit, nil
	case api.TreeNodeModeTree:
		return TreeNodeModeTree, nil
	default:
		return TreeNodeModeFile, fmt.Errorf("unknown rpc tree node mode: %d", m)
	}
}

func mapRenameDetails(c []api.PathRenameDetails) []*RenameDetails {
	renameDetailsList := make([]*RenameDetails, len(c))
	for i, detail := range c {
		renameDetailsList[i] = &RenameDetails{
			OldPath:         detail.OldPath,
			NewPath:         detail.Path,
			CommitShaBefore: detail.CommitSHABefore,
			CommitShaAfter:  detail.CommitSHAAfter,
		}
	}
	return renameDetailsList
}

func mapToSortOrder(o SortOrder) api.SortOrder {
	switch o {
	case SortOrderAsc:
		return api.SortOrderAsc
	case SortOrderDesc:
		return api.SortOrderDesc
	case SortOrderDefault:
		return api.SortOrderDefault
	default:
		// no need to error out - just use default for sorting
		return api.SortOrderDefault
	}
}

func mapHunkHeader(h *parser.HunkHeader) HunkHeader {
	return HunkHeader{
		OldLine: h.OldLine,
		OldSpan: h.OldSpan,
		NewLine: h.NewLine,
		NewSpan: h.NewSpan,
		Text:    h.Text,
	}
}

func mapDiffFileHeader(h *parser.DiffFileHeader) DiffFileHeader {
	return DiffFileHeader{
		OldName:    h.OldFileName,
		NewName:    h.NewFileName,
		Extensions: h.Extensions,
	}
}
