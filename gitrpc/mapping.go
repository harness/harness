// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"
)

func mapToRPCSortOrder(o SortOrder) rpc.SortOrder {
	switch o {
	case SortOrderAsc:
		return rpc.SortOrder_Asc
	case SortOrderDesc:
		return rpc.SortOrder_Desc
	case SortOrderDefault:
		return rpc.SortOrder_Default
	default:
		// no need to error out - just use default for sorting
		return rpc.SortOrder_Default
	}
}

func mapToRPCListBranchesSortOption(o BranchSortOption) rpc.ListBranchesRequest_SortOption {
	switch o {
	case BranchSortOptionName:
		return rpc.ListBranchesRequest_Name
	case BranchSortOptionDate:
		return rpc.ListBranchesRequest_Date
	case BranchSortOptionDefault:
		return rpc.ListBranchesRequest_Default
	default:
		// no need to error out - just use default for sorting
		return rpc.ListBranchesRequest_Default
	}
}

func mapToRPCListCommitTagsSortOption(o TagSortOption) rpc.ListCommitTagsRequest_SortOption {
	switch o {
	case TagSortOptionName:
		return rpc.ListCommitTagsRequest_Name
	case TagSortOptionDate:
		return rpc.ListCommitTagsRequest_Date
	case TagSortOptionDefault:
		return rpc.ListCommitTagsRequest_Default
	default:
		// no need to error out - just use default for sorting
		return rpc.ListCommitTagsRequest_Default
	}
}

func mapRPCBranch(b *rpc.Branch) (*Branch, error) {
	if b == nil {
		return nil, fmt.Errorf("rpc branch is nil")
	}

	var commit *Commit
	if b.GetCommit() != nil {
		var err error
		commit, err = mapRPCCommit(b.GetCommit())
		if err != nil {
			return nil, err
		}
	}

	return &Branch{
		Name:   b.Name,
		SHA:    b.Sha,
		Commit: commit,
	}, nil
}

func mapRPCCommitTag(t *rpc.CommitTag) (*CommitTag, error) {
	if t == nil {
		return nil, fmt.Errorf("rpc commit tag is nil")
	}

	var commit *Commit
	if t.GetCommit() != nil {
		var err error
		commit, err = mapRPCCommit(t.GetCommit())
		if err != nil {
			return nil, err
		}
	}

	var tagger *Signature
	if t.GetTagger() != nil {
		var err error
		tagger, err = mapRPCSignature(t.GetTagger())
		if err != nil {
			return nil, err
		}
	}

	return &CommitTag{
		Name:        t.Name,
		SHA:         t.Sha,
		IsAnnotated: t.IsAnnotated,
		Title:       t.Title,
		Message:     t.Message,
		Tagger:      tagger,
		Commit:      commit,
	}, nil
}

func mapRPCRenameDetails(c []*rpc.RenameDetails) []*RenameDetails {
	renameDetailsList := make([]*RenameDetails, len(c))
	for i, detail := range c {
		renameDetailsList[i] = &RenameDetails{
			OldPath:         detail.OldPath,
			NewPath:         detail.NewPath,
			CommitShaBefore: detail.CommitShaBefore,
			CommitShaAfter:  detail.CommitShaAfter,
		}
	}
	return renameDetailsList
}

func mapRPCCommit(c *rpc.Commit) (*Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("rpc commit is nil")
	}

	author, err := mapRPCSignature(c.GetAuthor())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc author: %w", err)
	}

	comitter, err := mapRPCSignature(c.GetCommitter())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc committer: %w", err)
	}

	return &Commit{
		SHA:       c.GetSha(),
		Title:     c.GetTitle(),
		Message:   c.GetMessage(),
		Author:    *author,
		Committer: *comitter,
	}, nil
}

func mapRPCSignature(s *rpc.Signature) (*Signature, error) {
	if s == nil {
		return nil, fmt.Errorf("rpc signature is nil")
	}

	identity, err := mapRPCIdentity(s.GetIdentity())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc identity: %w", err)
	}

	when := time.Unix(s.When, 0)

	return &Signature{
		Identity: identity,
		When:     when,
	}, nil
}

func mapRPCIdentity(id *rpc.Identity) (Identity, error) {
	if id == nil {
		return Identity{}, fmt.Errorf("rpc identity is nil")
	}

	return Identity{
		Name:  id.GetName(),
		Email: id.GetEmail(),
	}, nil
}

func mapRPCTreeNode(n *rpc.TreeNode) (TreeNode, error) {
	if n == nil {
		return TreeNode{}, fmt.Errorf("rpc tree node is nil")
	}

	nodeType, err := mapRPCTreeNodeType(n.GetType())
	if err != nil {
		return TreeNode{}, err
	}

	mode, err := mapRPCTreeNodeMode(n.GetMode())
	if err != nil {
		return TreeNode{}, err
	}

	return TreeNode{
		Type: nodeType,
		Mode: mode,
		SHA:  n.GetSha(),
		Name: n.GetName(),
		Path: n.GetPath(),
	}, nil
}

func mapRPCTreeNodeType(t rpc.TreeNodeType) (TreeNodeType, error) {
	switch t {
	case rpc.TreeNodeType_TreeNodeTypeBlob:
		return TreeNodeTypeBlob, nil
	case rpc.TreeNodeType_TreeNodeTypeCommit:
		return TreeNodeTypeCommit, nil
	case rpc.TreeNodeType_TreeNodeTypeTree:
		return TreeNodeTypeTree, nil
	default:
		return TreeNodeTypeBlob, fmt.Errorf("unknown rpc tree node type: %d", t)
	}
}

func mapRPCTreeNodeMode(m rpc.TreeNodeMode) (TreeNodeMode, error) {
	switch m {
	case rpc.TreeNodeMode_TreeNodeModeFile:
		return TreeNodeModeFile, nil
	case rpc.TreeNodeMode_TreeNodeModeExec:
		return TreeNodeModeExec, nil
	case rpc.TreeNodeMode_TreeNodeModeSymlink:
		return TreeNodeModeSymlink, nil
	case rpc.TreeNodeMode_TreeNodeModeCommit:
		return TreeNodeModeCommit, nil
	case rpc.TreeNodeMode_TreeNodeModeTree:
		return TreeNodeModeTree, nil
	default:
		return TreeNodeModeFile, fmt.Errorf("unknown rpc tree node mode: %d", m)
	}
}

func mapToRPCIdentityOptional(identity *Identity) *rpc.Identity {
	if identity == nil {
		return nil
	}

	return &rpc.Identity{
		Name:  identity.Name,
		Email: identity.Email,
	}
}

func mapToRPCTimeOptional(t *time.Time) int64 {
	if t == nil {
		return 0
	}

	return t.Unix()
}

func mapHunkHeader(h *rpc.HunkHeader) HunkHeader {
	return HunkHeader{
		OldLine: int(h.OldLine),
		OldSpan: int(h.OldSpan),
		NewLine: int(h.NewLine),
		NewSpan: int(h.NewSpan),
		Text:    h.Text,
	}
}

func mapDiffFileHeader(h *rpc.DiffFileHeader) DiffFileHeader {
	return DiffFileHeader{
		OldName:    h.OldFileName,
		NewName:    h.NewFileName,
		Extensions: h.Extensions,
	}
}
